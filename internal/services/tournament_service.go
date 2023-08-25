package services

import (
	"encoding/json"
	"goodBlast-backend/internal/models"
	"goodBlast-backend/internal/repositories"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

type TournamentService struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	dynamoDBRepo *repositories.DynamoDBRepository
}

func NewTournamentService(conn *amqp.Connection) (*TournamentService, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	dynamoDBRepo, err := repositories.NewDynamoDBRepository()
	if err != nil {
		return nil, err
	}

	return &TournamentService{
		conn:        conn,
		channel:     channel,
		dynamoDBRepo: dynamoDBRepo,
	}, nil
}

func (ts *TournamentService) Start() {
	defer ts.conn.Close()
	defer ts.channel.Close()

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
		"tournamentQueue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	msgs, err := ts.channel.Consume(
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
		case "StartTournament":
			ts.HandleStartTournament(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "EnterTournament":
			ts.HandleEnterTournament(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "UpdateScore":
			ts.HandleUpdateScore(msg.Body, msg.ReplyTo, msg.CorrelationId)
		default:
			log.Printf("Unknown action: %s", action)
		}
	}
}

func (ts *TournamentService) Stop() {
	log.Println("Stopping tournament service...")
	if err := ts.channel.Close(); err != nil {
		log.Printf("Error closing channel: %v", err)
	}
}


func (ts *TournamentService) HandleStartTournament(data []byte, replyTo string, correlationID string) {
	var requestData struct {
		Action string `json:"action"`
	}

	err := json.Unmarshal(data, &requestData)
	if err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		return
	}

	// Generate a unique tournament ID using UUID
	tournamentID := uuid.New().String()

	// Get the current time
	currentTime := time.Now().UTC()

	// Calculate the end time as 23 hours and 59 minutes later from the current time
	endTime := currentTime.Add(23*time.Hour + 59*time.Minute)

	tournament := models.Tournament{
		TournamentID: tournamentID,
		StartTime:    currentTime,
		EndTime:      endTime,
		NumRegisteredUsers: 0,
		Finished: false,
	}

	// Create the tournament in the DynamoDB repository
	err = ts.dynamoDBRepo.CreateTournament(&tournament)
	if err != nil {
		log.Printf("Failed to create tournament: %v", err)
		sendResponse(ts.channel, replyTo, correlationID, "StartTournamentResponse", "Failed to start tournament")
		return
	}

	sendResponse(ts.channel, replyTo, correlationID, "StartTournamentResponse", struct {
		TournamentID string `json:"tournament_id"`
		StartTime    string `json:"start_time"`
		EndTime      string `json:"end_time"`
		NumRegisteredUsers int `json:"num_registered_users"`
		Finished 	 bool `json:"finished"`
	}{
		TournamentID: tournamentID,
		StartTime:    tournament.StartTime.Format(time.RFC3339),
		EndTime:      tournament.EndTime.Format(time.RFC3339),
		NumRegisteredUsers: tournament.NumRegisteredUsers,
		Finished: tournament.Finished,
	})

	log.Printf("Tournament started: %+v", tournament)
}

func (ts *TournamentService) HandleEnterTournament(data []byte, replyTo string, correlationID string) {
	var requestData struct {
		Action       string `json:"action"`
		Username     string `json:"username"`
		TournamentID string `json:"tournament_id"`
	}

	err := json.Unmarshal(data, &requestData)
	if err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		return
	}

	groupID, err := ts.dynamoDBRepo.RegisterToTournament(requestData.Username)

	if err != nil {
		log.Printf("Failed to enter tournament: %v", err)
		sendResponse(ts.channel, replyTo, correlationID, "EnterTournamentResponse", struct {
			Error string `json:"error"`
		}{
			Error: "Failed to enter tournament",
		})
		return
	}

	if groupID == -2 {
		log.Printf("User is already registered for the tournament: %v", requestData.Username)
		sendResponse(ts.channel, replyTo, correlationID, "EnterTournamentResponse", struct {
            Error string `json:"error"`
        }{
            Error: "User is already registered for the tournament",
        })
		return
	};

	if groupID == -3 {
		log.Printf("User has not enough coins to enter the tournament: %v", requestData.Username)
		sendResponse(ts.channel, replyTo, correlationID, "EnterTournamentResponse", struct {
            Error string `json:"error"`
        }{
            Error: "User has not enough coins to enter the tournament",
        })
		return
	}

	if groupID == -4 {
		log.Printf("User has not enough progress level to enter the tournament: %v", requestData.Username)
		sendResponse(ts.channel, replyTo, correlationID, "EnterTournamentResponse", struct {
            Error string `json:"error"`
        }{
            Error: "User has not enough progress level to enter the tournament",
        })
		return
	}
		
	sendResponse(ts.channel, replyTo, correlationID, "EnterTournamentResponse", struct {
		GroupID int `json:"group_id"`
	}{
		GroupID: groupID,
	})

	log.Printf("User entered tournament: %+v", requestData)
	return
}


func (ts *TournamentService) HandleUpdateScore(data []byte, replyTo string, correlationID string) {
    var requestData struct {
        Action   string `json:"action"`
        Username string `json:"username"`
    }

    err := json.Unmarshal(data, &requestData)
    if err != nil {
        log.Printf("Failed to unmarshal data: %v", err)
        return
    }

    latestTournamentID, err := ts.dynamoDBRepo.GetLatestTournamentForUser(requestData.Username)
    if err != nil {
        sendResponse(ts.channel, replyTo, correlationID, "UpdateScoreResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Failed to get latest tournament for user",
        })
        return
    }

    isTournamentActive, err := ts.dynamoDBRepo.IsTournamentActive(latestTournamentID)
    if err != nil {
        sendResponse(ts.channel, replyTo, correlationID, "UpdateScoreResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Failed to check if tournament is active",
        })
        return
    }

    if isTournamentActive {
        progressLevel, coins, score, err := ts.dynamoDBRepo.IncrementUserScoreInTournament(requestData.Username, latestTournamentID)
        if err != nil {
			log.Printf("Failed to increment user score in tournament: %v", err)
            sendResponse(ts.channel, replyTo, correlationID, "UpdateScoreResponse", struct {
                Error string `json:"error"`
            }{
                Error: "Failed to increment user score in tournament",
            })
            return
        }
        
        sendResponse(ts.channel, replyTo, correlationID, "UpdateScoreResponse", struct {
            Progress_Level int `json:"progress_level"`
            Coins         int `json:"coins"`
            Score         int `json:"score"`
        }{
            Progress_Level: progressLevel,
            Coins:         coins,
            Score:         score,
        })
    }
}
