package services

import (
	"encoding/json"
	"goodBlast-backend/internal/models"
	"goodBlast-backend/internal/repositories"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

type TournamentService struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	dynamoDBRepo *repositories.DynamoDBRepository
}

// Create a new tournament service
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

// Initialize the tournament service
func (ts *TournamentService) Start() {
	defer ts.conn.Close()
	defer ts.channel.Close()

    // Establish RabbitMQ connection
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

    //Start RabbitMQ channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

    // Declare the "tournamentQueue" queue
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

    // Start consuming messages
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

    // Handle messages received on the "tournamentQueue" queue
	for msg := range msgs {
		action, ok := msg.Headers["action"].(string)
		if !ok {
			log.Printf("Invalid or missing action field in headers")
			continue
		}

        // Handle the message based on the action
		switch action {
		case "StartTournament":
			ts.HandleStartTournament(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "EnterTournament":
			ts.HandleEnterTournament(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "UpdateScore":
			ts.HandleUpdateScore(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "ClaimReward":
			ts.HandleClaimReward(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "EndTournament":
			ts.EndTournament(msg.Body, msg.ReplyTo, msg.CorrelationId)
		default:
			log.Printf("Unknown action: %s", action)
		}
	}
}

// Stop the tournament service
func (ts *TournamentService) Stop() {
	log.Println("Stopping tournament service...")
	if err := ts.channel.Close(); err != nil {
		log.Printf("Error closing channel: %v", err)
	}
}

// Start a new tournament
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
        LatestGroupID: 0,
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

// Enter a tournament with the given username and tournament ID
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

    // Atomically increment the number of registered users in the tournament
    // and get the latest group ID to assign to the user
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

    // Check if the user is already registered for the tournament
	if groupID == -2 {
		log.Printf("User is already registered for the tournament: %v", requestData.Username)
		sendResponse(ts.channel, replyTo, correlationID, "EnterTournamentResponse", struct {
            Error string `json:"error"`
        }{
            Error: "User is already registered for the tournament",
        })
		return
	};

    // Check if the user has enough coins to enter the tournament
	if groupID == -3 {
		log.Printf("User has not enough coins to enter the tournament: %v", requestData.Username)
		sendResponse(ts.channel, replyTo, correlationID, "EnterTournamentResponse", struct {
            Error string `json:"error"`
        }{
            Error: "User has not enough coins to enter the tournament",
        })
		return
	}

    // Check if the user has enough progress level to enter the tournament
	if groupID == -4 {
		log.Printf("User has not enough progress level to enter the tournament: %v", requestData.Username)
		sendResponse(ts.channel, replyTo, correlationID, "EnterTournamentResponse", struct {
            Error string `json:"error"`
        }{
            Error: "User has not enough progress level to enter the tournament",
        })
		return
	}

    // Check if the user claimed reward for the previous tournament
    DidUserClaimReward, err := ts.dynamoDBRepo.DidUserClaimReward(requestData.Username)
    if err != nil {
        log.Printf("Failed to check if user claimed reward: %v", err)
        sendResponse(ts.channel, replyTo, correlationID, "EnterTournamentResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Failed to check if user claimed reward",
        })
        return
    }
    if DidUserClaimReward {
        log.Printf("User already claimed reward: %v", requestData.Username)
        sendResponse(ts.channel, replyTo, correlationID, "EnterTournamentResponse", struct {
            Error string `json:"error"`
        }{
            Error: "User did not claim reward for previous tournament",
        })
        return
    }

    // Get the latest tournament ID for the user
    latestTournamentID, err := ts.dynamoDBRepo.GetLatestTournamentForUser(requestData.Username)
    if err != nil {
        sendResponse(ts.channel, replyTo, correlationID, "UpdateScoreResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Failed to get latest tournament for user",
        })
        return
    }

    //Create the leaderboard name
    newTournamentID := latestTournamentID + ":" + strconv.Itoa(groupID)

    // Send EnterLeaderboardGroup action to LeaderboardService
    action := "EnterLeaderboardGroup"
    messageData := map[string]interface{}{
        "action"  :  action,
        "grpup_id": groupID,
        "leaderboard_name": newTournamentID,
        "username": requestData.Username,
        "initial_score": 0,
    }

    // Publish the message to the "leaderboardQueue" with the publishToRabbitMQ function
    publishToRabbitMQ(ts.channel, "leaderboardQueue", action, messageData, replyTo, correlationID)
    log.Printf("Sent enterLeaderboardGroup action to LeaderboardService")

    // Update the user's latest group ID
    ts.dynamoDBRepo.UpdateUserField(requestData.Username, "latest_group_id", groupID)

	sendResponse(ts.channel, replyTo, correlationID, "EnterTournamentResponse", struct {
		GroupID int `json:"group_id"`
	}{
		GroupID: groupID,
	})

	log.Printf("User entered tournament: %+v", requestData)
	return
}

// Update the score of a user in the tournament
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

    // Get the latest tournament ID for the user
    latestTournamentID, err := ts.dynamoDBRepo.GetLatestTournamentForUser(requestData.Username)
    if err != nil {
        sendResponse(ts.channel, replyTo, correlationID, "UpdateScoreResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Failed to get latest tournament for user",
        })
        return
    }

    // Check if the tournament is active
    isTournamentActive, err := ts.dynamoDBRepo.IsTournamentActive(latestTournamentID)
    if err != nil {
        sendResponse(ts.channel, replyTo, correlationID, "UpdateScoreResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Failed to check if tournament is active",
        })
        return
    }

    // If the tournament is active, increment the user's score in the tournament
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

        // Get the latest group ID for the user
        latestGroupID, err := ts.dynamoDBRepo.GetLatestGroupIdForUser(requestData.Username)

        // Check if the user is not assigned to any group
        if latestGroupID == -2 {
            log.Printf("Failed to get latest group id for user: %v", err)
            sendResponse(ts.channel, replyTo, correlationID, "UpdateScoreResponse", struct {
                Error string `json:"error"`
            }{
                Error: "User is not assigned to any group",
            })
            return
        }

        if err != nil {
            sendResponse(ts.channel, replyTo, correlationID, "UpdateScoreResponse", struct {
                Error string `json:"error"`
            }{
                Error: "Failed to get latest group id for user",
            })
            return
        }

        // Create the leaderboard name
        newLeaderboardName := latestTournamentID + ":" + strconv.Itoa(latestGroupID)

        // Send IncrementGroupScore action to LeaderboardService
        action := "IncrementGroupScore" // Define the action
        messageData := map[string]interface{}{
            "action":       action,
            "group_id": latestGroupID,
            "leaderboard_name": newLeaderboardName,
            "username": requestData.Username,
        }
        // Publish the message to the "leaderboardQueue" with the publishToRabbitMQ function
        log.Printf("messageData: %v", messageData)
        publishToRabbitMQ(ts.channel, "leaderboardQueue", action, messageData, replyTo, correlationID)

        log.Printf("Sent incrementGroupScore action to LeaderboardService")
        
        sendResponse(ts.channel, replyTo, correlationID, "UpdateScoreResponse", struct {
            Progress_Level int `json:"progress_level"`
            Coins         int `json:"coins"`
            Score         int `json:"score"`
        }{
            Progress_Level: progressLevel,
            Coins:         coins,
            Score:         score,
        })
    } else {
        sendResponse(ts.channel, replyTo, correlationID, "UpdateScoreResponse", struct {
                Error string `json:"error"`
        }{
                Error: "User is not in any active tournament",
         })
        return 
    }

}

// End the latest tournament
func (ts *TournamentService) EndTournament(data []byte, replyTo string, correlationID string) {
    var requestData struct {
        Action string `json:"action"`
    }

    err := json.Unmarshal(data, &requestData)
    if err != nil {
        log.Printf("Failed to unmarshal data: %v", err)
        return
    }


    // End the latest tournament
    tournamentID, err := ts.dynamoDBRepo.EndLatestTournament()
    if err != nil {
        log.Printf("Failed to end latest tournament: %v", err)
        sendResponse(ts.channel, replyTo, correlationID, "EndTournamentResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Failed to end latest tournament",
        })
        return
    }

    // Get the ranked players for the latest tournament
    rankedPlayers, err := ts.dynamoDBRepo.FindRankedPlayersForLatestTournament()
    if err != nil {
        log.Printf("Failed to find ranked players: %v", err)
        sendResponse(ts.channel, replyTo, correlationID, "EndTournamentResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Failed to find ranked players",
        })
        return
    }


    // Send DeleteLeaderboard action to LeaderboardService
	action := "DeleteLeaderboard" // Define the action
    messageData := map[string]interface{}{
        "action":       action,
        "tournament_id": tournamentID, // Include the tournamentID here
    }
    // Publish the message to the "leaderboardQueue" with the publishToRabbitMQ function
    publishToRabbitMQ(ts.channel, "leaderboardQueue", action, messageData, replyTo, correlationID)

    log.Printf("Sent deleteLeaderboard action to LeaderboardService")


    sendResponse(ts.channel, replyTo, correlationID, "EndTournamentResponse", struct {
        RankedPlayers []models.UserInTournament `json:"ranked_players"`
    }{
        RankedPlayers: rankedPlayers,
    })
}

// Claim the reward for the latest tournament for the given username
func (ts *TournamentService) HandleClaimReward(data []byte, replyTo string, correlationID string) {
	var requestData struct {
        Action   string `json:"action"`
        Username string `json:"username"`
    }

    err := json.Unmarshal(data, &requestData)
    if err != nil {
        log.Printf("Failed to unmarshal data: %v", err)
        return
    }

    // Get the username from the request data
    username := requestData.Username
    // Get the latest tournament ID for the user
    latestTournamentID, err := ts.dynamoDBRepo.GetLatestTournamentForUser(username)
    if err != nil {
        sendResponse(ts.channel, replyTo, correlationID, "ClaimRewardResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Failed to get latest tournament for user",
        })
        return
    }

    if latestTournamentID == "" {
        // User has not joined any tournament yet
        sendResponse(ts.channel, replyTo, correlationID, "ClaimRewardResponse", struct {
            Error string `json:"error"`
        }{
            Error: "User has not joined any tournament yet",
        })
        return
    }

    // Check if the tournament is finished
    isTournamentFinished, err := ts.dynamoDBRepo.IsTournamentFinished(latestTournamentID)
    if err != nil {
        sendResponse(ts.channel, replyTo, correlationID, "ClaimRewardResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Failed to check if tournament is finished",
        })
        return
    }

    // If the tournament is not finished, return an error
    if !isTournamentFinished {
        sendResponse(ts.channel, replyTo, correlationID, "ClaimRewardResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Reward cannot be claimed yet. Tournament is not finished",
        })
        return
    }

    // Get the user's entry in the tournament
    userInTournament, err := ts.dynamoDBRepo.GetUserInTournamentByUsernameAndTournamentID(username, latestTournamentID)
    if err != nil {
        sendResponse(ts.channel, replyTo, correlationID, "ClaimRewardResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Failed to get user's entry in the tournament",
        })
        return
    }

    // Check if the user's entry is not found in the tournament
    if userInTournament == nil {
        sendResponse(ts.channel, replyTo, correlationID, "ClaimRewardResponse", struct {
            Error string `json:"error"`
        }{
            Error: "User's entry not found in the tournament",
        })
        return
    }

    // Check if the user's reward is already claimed
    if userInTournament.Claimed {
        sendResponse(ts.channel, replyTo, correlationID, "ClaimRewardResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Reward is already claimed",
        })
        return
    }

    // Calculate the reward amount based on the user's rank
    var rewardAmount int
    switch userInTournament.Rank {
    case 1:
        rewardAmount = 5000
    case 2:
        rewardAmount = 3000
    case 3:
        rewardAmount = 2000
    case 4:
        rewardAmount = 1000
    default:
        rewardAmount = 0
    }

	// Get the user's current coin balance
	user, err := ts.dynamoDBRepo.GetUserByUsername(username)
	if err != nil {
		sendResponse(ts.channel, replyTo, correlationID, "ClaimRewardResponse", struct {
			Error string `json:"error"`
		}{
			Error: "Failed to get user's data",
		})
		return
	}
	if err != nil {
		sendResponse(ts.channel, replyTo, correlationID, "ClaimRewardResponse", struct {
			Error string `json:"error"`
		}{
			Error: "Failed to get user's data",
		})
		return
	}

	// Update user's coin balance and mark the reward as claimed
	err = ts.dynamoDBRepo.UpdateUserField(username, "coins", user.Coins+rewardAmount)
	if err != nil {
		sendResponse(ts.channel, replyTo, correlationID, "ClaimRewardResponse", struct {
			Error string `json:"error"`
		}{
			Error: "Failed to update user's coin balance",
		})
		return
	}

    // Mark the reward as claimed
    err = ts.dynamoDBRepo.UpdateUserInTournamentClaimed(username, latestTournamentID, true)
    if err != nil {
        sendResponse(ts.channel, replyTo, correlationID, "ClaimRewardResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Failed to mark reward as claimed",
        })
        return
    }

    sendResponse(ts.channel, replyTo, correlationID, "ClaimRewardResponse", struct {
        Success       bool `json:"success"`
        RewardClaimed int  `json:"reward_claimed"`
    }{
        Success:       true,
        RewardClaimed: rewardAmount,
    })
}

