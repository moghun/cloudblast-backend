package services

import (
	"encoding/json"
	"log"

	"goodBlast-backend/internal/repositories"

	"github.com/streadway/amqp"
)

type LeaderboardService struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	redisRepo   *repositories.RedisRepo
}

func NewLeaderboardService(conn *amqp.Connection, redisAddr string) (*LeaderboardService, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Initialize Redis repository
	redisRepo, err := repositories.NewRedisRepo(redisAddr)
	if err != nil {
		return nil, err
	}

	return &LeaderboardService{
		conn:        conn,
		channel:     channel,
		redisRepo:   redisRepo,
	}, nil
}

func (ls *LeaderboardService) Start() {
	defer ls.conn.Close()
	defer ls.channel.Close()

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"leaderboardQueue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	msgs, err := ls.channel.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	for msg := range msgs {
		action, ok := msg.Headers["action"].(string)
		if !ok {
			log.Printf("Invalid or missing action field in headers")
			continue
		}

		switch action {
		case "createLeaderboard":
			ls.HandleCreateLeaderboard(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "deleteLeaderboard":
			ls.HandleDeleteLeaderboard(msg.Body, msg.ReplyTo, msg.CorrelationId)
		default:
			log.Printf("Unknown action: %s", action)
		}
	}
}


func (ls *LeaderboardService) Stop() {
	log.Println("Stopping leaderboard service...")
	if err := ls.channel.Close(); err != nil {
		log.Printf("Error closing channel: %v", err)
	}
}

func (ls *LeaderboardService) HandleCreateLeaderboard(data []byte, replyTo string, correlationID string) {
	var requestData struct {
		Action          string `json:"action"`
		LeaderboardName string `json:"tournament_id"`
	}

	err := json.Unmarshal(data, &requestData)
	if err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		return
	}

	// Create a new leaderboard in Redis
	err = ls.redisRepo.CreateLeaderboard(requestData.LeaderboardName)
	if err != nil {
		log.Printf("Error creating leaderboard: %v", err)
		return
	}
	log.Printf("Created leaderboard %s", requestData.LeaderboardName)
}

func (ls *LeaderboardService) HandleDeleteLeaderboard(data []byte, replyTo string, correlationID string) {
	var requestData struct {
		Action          string `json:"action"`
		LeaderboardName string `json:"tournament_id"`
	}

	err := json.Unmarshal(data, &requestData)
	if err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		return
	}

	// Delete the specified leaderboard from Redis
	err = ls.redisRepo.DeleteLeaderboard(requestData.LeaderboardName)
	if err != nil {
		log.Printf("Error deleting leaderboard: %v", err)
		return
	}
	log.Printf("Deleted leaderboard %s", requestData.LeaderboardName)
}