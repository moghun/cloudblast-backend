package services

import (
	"encoding/json"
	"log"
	"strconv"

	"goodBlast-backend/internal/repositories"

	"github.com/streadway/amqp"
)

type LeaderboardService struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	dynamoDBRepo *repositories.DynamoDBRepository
	redisRepo   *repositories.RedisRepo
}

func NewLeaderboardService(conn *amqp.Connection, redisAddr string) (*LeaderboardService, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	dynamoDBRepo, err := repositories.NewDynamoDBRepository()
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
		dynamoDBRepo: dynamoDBRepo,
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
		case "DeleteLeaderboard":
			ls.HandleDeleteLeaderboard(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "EnterLeaderboardGroup":
			ls.HandleEnterLeaderboardGroup(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "IncrementGroupScore":
			ls.HandleIncrementGroupScore(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "GetGroupUserRank":
			ls.HandleGetGroupUserRank(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "GetGroupLeaderboardWithRanks":
			ls.HandleGetGroupLeaderboardWithRanks(msg.Body, msg.ReplyTo, msg.CorrelationId)
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
	err = ls.redisRepo.DeleteLeaderboards(requestData.LeaderboardName)
	if err != nil {
		log.Printf("Error deleting leaderboard: %v", err)
		return
	}
	log.Printf("Deleted leaderboard %s", requestData.LeaderboardName)
}

func (ls *LeaderboardService) HandleEnterLeaderboardGroup(data []byte, replyTo string, correlationID string) {
	var requestData struct {
		Action          string `json:"action"`
		GroupID         string `json:"group_id"`
		LeaderboardName string `json:"leaderboard_name"`
		Username        string `json:"username"`
		InitialScore    int    `json:"initial_score"`
	}

	err := json.Unmarshal(data, &requestData)
	if err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		return
	}

	// Enter the leaderboard group for the specified user with an initial score
	err = ls.redisRepo.EnterLeaderboardGroup(requestData.LeaderboardName, requestData.Username, requestData.InitialScore)
	if err != nil {
		log.Printf("Error entering leaderboard group: %v", err)
		return
	}
	log.Printf("Entered leaderboard group: %s - %s - %s - %d", requestData.GroupID, requestData.LeaderboardName, requestData.Username, requestData.InitialScore)
}

func (ls *LeaderboardService) HandleIncrementGroupScore(data []byte, replyTo string, correlationID string) {
	var requestData struct {
		Action         string `json:"action"`
		GroupID        int `json:"group_id"`
		LeaderboardName string `json:"leaderboard_name"`
		Username       string `json:"username"`
	}

	err := json.Unmarshal(data, &requestData)
	if err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		return
	}

	// Increment the user's score in the specified leaderboard
	err = ls.redisRepo.IncrementGroupScore(requestData.LeaderboardName, requestData.Username)
	if err != nil {
		log.Printf("Error incrementing user's score: %v", err)
		return
	}

	log.Printf("Incremented score for user %s in leaderboard %s", requestData.Username, requestData.LeaderboardName)
}

func (ls *LeaderboardService) HandleGetGroupUserRank(data []byte, replyTo string, correlationID string) {
	var requestData struct {
		Action         string `json:"action"`
		Username       string `json:"username"`
	}
	err := json.Unmarshal(data, &requestData)
	if err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		return
	}

	latestTournamentID, err := ls.dynamoDBRepo.GetLatestTournamentForUser(requestData.Username)
	if err != nil {
		log.Printf("Error getting latest tournament for user: %v", err)
		return
	}

	latestGroupID, err := ls.dynamoDBRepo.GetLatestGroupIdForUser(requestData.Username)
	if err != nil {
		log.Printf("Error getting latest group ID for user: %v", err)
		return
	}


	newLeaderboardName := latestTournamentID + ":" + strconv.Itoa(latestGroupID)
	// Get the user's rank in the specified leaderboard
	rank, err := ls.redisRepo.GetGroupUserRank(newLeaderboardName, requestData.Username)
	if err != nil {
		log.Printf("Error getting user's rank: %v", err)
		return
	}

	// Publish the user's rank as a response
	sendResponse(ls.channel, replyTo, correlationID,"GetGroupUserRankResponse", struct {
		Rank int64 `json:"rank"`
	}{
		Rank: rank,
	})

	log.Printf("User %s rank in leaderboard %s: %d", requestData.Username, newLeaderboardName, rank)
}

func (ls *LeaderboardService) HandleGetGroupLeaderboardWithRanks(data []byte, replyTo string, correlationID string) {
	var requestData struct {
		Action         string `json:"action"`
		Username string `json:"username"`
	}

	err := json.Unmarshal(data, &requestData)
	if err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		return
	}

	latestTournamentID, err := ls.dynamoDBRepo.GetLatestTournamentForUser(requestData.Username)
	if err != nil {
		log.Printf("Error getting latest tournament for user: %v", err)
		return
	}

	latestGroupID, err := ls.dynamoDBRepo.GetLatestGroupIdForUser(requestData.Username)
	if err != nil {
		log.Printf("Error getting latest group ID for user: %v", err)
		return
	}

	
	newLeaderboardName := latestTournamentID + ":" + strconv.Itoa(latestGroupID)
	log.Printf("Getting leaderboard for group %s", newLeaderboardName)

	// Get the leaderboard with ranks for the specified group and leaderboard name
	leaderboard, err := ls.redisRepo.GetGroupLeaderboardWithRanks(newLeaderboardName, 0, 34)
	if err != nil {
		log.Printf("Error getting group leaderboard: %v", err)
		return
	}

	sendResponse(ls.channel, replyTo, correlationID,"GetGroupLeaderboardWithRanksResponse", leaderboard)

}
