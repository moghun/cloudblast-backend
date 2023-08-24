package services

import (
	"encoding/json"
	"goodBlast-backend/internal/repositories"
	"log"

	"github.com/streadway/amqp"
)

type UserService struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	dynamoDBRepo *repositories.DynamoDBRepository
}

func NewUserService(conn *amqp.Connection) (*UserService, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	dynamoDBRepo, err := repositories.NewDynamoDBRepository()
	if err != nil {
		return nil, err
	}

	return &UserService{
		conn:        conn,
		channel:     channel,
		dynamoDBRepo: dynamoDBRepo,
	}, nil
}

func (uh *UserService) Start() {
	defer uh.conn.Close()
	defer uh.channel.Close()

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
		"userQueue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	msgs, err := uh.channel.Consume(
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
		case "SearchUser":
			uh.HandleSearchUser(msg.Body, msg.ReplyTo, msg.CorrelationId)
		default:
			log.Printf("Unknown action: %s", action)
		}
	}
}

func (uh *UserService) Stop() {
	log.Println("Stopping user_Service service...")
	if err := uh.channel.Close(); err != nil {
		log.Printf("Error closing channel: %v", err)
	}
}

func (uh *UserService) HandleUpdateUser(b []byte) {
	panic("unimplemented")
}

func (uh *UserService) HandleCreateUser(b []byte) {
	panic("unimplemented")
}

func (uh *UserService) HandleSearchUser(data []byte, replyTo string, correlationID string) {
	var requestData struct {
		Action   string `json:"action"`
		Username string `json:"username"`
	}

	err := json.Unmarshal(data, &requestData)
	if err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		return
	}

	user, err := uh.dynamoDBRepo.GetUserByUsername(requestData.Username)
	if err != nil {
		log.Printf("Error fetching user: %v", err)
		return
	}

	responseData := struct {
		Action string      `json:"action"`
		User   interface{} `json:"user"`
	}{
		Action: "SearchUserResponse",
		User:   user,
	}

	responseDataJSON, err := json.Marshal(responseData)
	if err != nil {
		log.Fatalf("Failed to encode response data to JSON: %v", err)
	}

	// Publish response to the replyTo queue
	err = uh.channel.Publish(
		"",
		replyTo,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			Body:          responseDataJSON,
			Headers: amqp.Table{
				"action": "SearchUserResponse",
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to publish a response message: %v", err)
	}

	log.Printf("User data: %+v", user)
}