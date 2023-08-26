package services

import (
	"encoding/json"
	"goodBlast-backend/internal/auth"
	"goodBlast-backend/internal/models"
	"goodBlast-backend/internal/repositories"
	"log"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	dynamoDBRepo *repositories.DynamoDBRepository
}

// Create a new user service
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
// HashPassword hashes the given password using bcrypt for Create User
func HashPassword(password string) (string, error) {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(hashedPassword), nil
}

// CheckPasswordHash checks if the given password matches the hashed password
func CheckPasswordHash(password, hashedPassword string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
    return err == nil
}

// Initialize the user service
func (uh *UserService) Start() {
	defer uh.conn.Close()
	defer uh.channel.Close()

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

    // Declare the user queue
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

    // Register a consumer
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

    // Handle messages
	for msg := range msgs {
		action, ok := msg.Headers["action"].(string)
		if !ok {
			log.Printf("Invalid or missing action field in headers")
			continue
		}

        // Handle the message based on the action
		switch action {
		case "Login":
			uh.HandleLogin(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "SearchUser":
			uh.HandleSearchUser(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "CreateUser":
			uh.HandleCreateUser(msg.Body, msg.ReplyTo, msg.CorrelationId)
		case "UpdateProgress":
			uh.HandleUpdateProgress(msg.Body, msg.ReplyTo, msg.CorrelationId)
        case "GetCountryLeaderboard":
            uh.HandleGetCountryLeaderboard(msg.Body, msg.ReplyTo, msg.CorrelationId)
        case "GetGlobalLeaderboard":
            uh.HandleGetGlobalLeaderboard(msg.Body, msg.ReplyTo, msg.CorrelationId)
		default:
			log.Printf("Unknown action: %s", action)
		}
	}
}

// Stop the user service
func (uh *UserService) Stop() {
	log.Println("Stopping user service service...")
	if err := uh.channel.Close(); err != nil {
		log.Printf("Error closing channel: %v", err)
	}
}

// Search for a user by username
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

    if user == nil {

        sendResponse(uh.channel, replyTo, correlationID, "SearchUserResponse", struct {
            Action string `json:"action"`
            Error  string `json:"error"`
        }{
            Action: "SearchUserResponse",
            Error:  "User not found",
        })
        return
    }

    sendResponse(uh.channel, replyTo, correlationID, "SearchUserResponse",struct {
        ID string `json:"uid"`
        Username string `json:"username"`
        Country string `json:"country"`
        Progress_Level int `json:"progress_level"`
        Coins int `json:"coins"`
        Latest_Tournament_ID string `json:"latest_tournament_id"`
        Latest_Group_ID int `json:"latest_group_id"`
    }{
        ID:   user.ID,
        Username: user.Username,
        Country: user.Country,
        Progress_Level: user.Progress_Level,
        Coins: user.Coins,
        Latest_Tournament_ID: user.Latest_Tournament_ID,
        Latest_Group_ID: user.Latest_Group_ID,
    })
}


// Create a new user 
func (uh *UserService) HandleCreateUser(data []byte, replyTo string, correlationID string) {
    var requestData struct {
        Action   string `json:"action"`
        Username string `json:"username"`
        Password string `json:"password"`
        Country string `json:"country"`
    }

    err := json.Unmarshal(data, &requestData)
    if err != nil {
        log.Printf("Failed to unmarshal data: %v", err)
        return
    }

    // Check if the username already exists
    existingUser, err := uh.dynamoDBRepo.GetUserByUsername(requestData.Username)
    if err != nil {
        log.Printf("Error fetching user: %v", err)
        return
    }

    // If the user already exists, return an error
    if existingUser != nil {
        sendResponse(uh.channel, replyTo, correlationID, "CreateUserResponse", struct {
            Error string `json:"error"`
        }{
            Error: "Username already exists",
        })
        return
    }

    // Hash the password
    hashedPassword, err := HashPassword(requestData.Password)
    if err != nil {
        log.Printf("Failed to hash password: %v", err)
        return
    }

    // Generate a unique identifier
    uniqueID := uuid.New().String()

    // Create the user object
    user := models.User{
        ID:       uniqueID,
        Username: requestData.Username,
        Password: hashedPassword,
        Country: requestData.Country,
        Progress_Level:    1,
        Coins:    100,
        Latest_Tournament_ID: "",
        Latest_Group_ID: -1,
    }

    // Create user in the database
    err = uh.dynamoDBRepo.CreateUser(&user)
    if err != nil {
        log.Printf("Error creating user: %v", err)
        return
    }

    sendResponse(uh.channel, replyTo, correlationID, "CreateUserResponse", struct {
        UserID string `json:"user_id"`
    }{
        UserID: uniqueID,
    })

    log.Printf("User created: %+v", user)
}

// Login a user
func (uh *UserService) HandleLogin(data []byte, replyTo string, correlationID string) {
    var requestData struct {
        Action   string `json:"action"`
        Username string `json:"username"`
        Password string `json:"password"`
    }

    err := json.Unmarshal(data, &requestData)
    if err != nil {
        log.Printf("Failed to unmarshal data: %v", err)
        return
    }

    // Get the user from the database 
    // Check if the username exists
    user, err := uh.dynamoDBRepo.GetUserByUsername(requestData.Username)
    if err != nil {
        log.Printf("Error fetching user: %v", err)
        return
    }
    if err != nil {
        log.Printf("Error fetching user: %v", err)
        return
    }

    // If the user does not exist, return an error
    if user == nil {
        sendResponse(uh.channel, replyTo, correlationID, "LoginResponse", struct {
            Token  string `json:"token"`
            Success bool `json:"success"`
        }{
            Token:  "",
            Success: false,
        })
        return
    }

    // Check if the password is correct
    if !CheckPasswordHash(requestData.Password, user.Password) {
        sendResponse(uh.channel, replyTo, correlationID, "LoginResponse", struct {
            Token  string `json:"token"`
            Success bool `json:"success"`
        }{
            Token:  "",
            Success: false,
        })
        return
    }

    // Generate a JWT token
    token, err := auth.CreateToken(user.Username)
    if err != nil {
        log.Fatalf("Failed to generate JWT token: %v", err)
        return
    }

    sendResponse(uh.channel, replyTo, correlationID, "LoginResponse", struct {
        Token  string `json:"token"`
        Success bool `json:"success"`
    }{
        Token:  token,
        Success: true,
    })
}

// Update a user's Progress_Level
func (uh *UserService) HandleUpdateProgress(data []byte, replyTo string, correlationID string) {
    var requestData struct {
        Action   string `json:"action"`
        Username string `json:"username"`
    }

    err := json.Unmarshal(data, &requestData)
    if err != nil {
        log.Printf("Failed to unmarshal data: %v", err)
        return
    }

    // Get the user from the database
    user, err := uh.dynamoDBRepo.GetUserByUsername(requestData.Username)
    if err != nil {
        log.Printf("Error fetching user: %v", err)
        return
    }


    if err != nil {
        log.Printf("Error fetching user: %v", err)
        return
    }

    // If the user does not exist, return an error
    if user == nil {
        sendResponse(uh.channel, replyTo, correlationID, "UpdateProgressResponse", struct {
            Progress_Level int `json:"progress_level"`
            Coins int `json:"coins"`
        }{
            Progress_Level: 0,
            Coins: 0,
        })
        return
    }

    // Update the user's progress
    err = uh.dynamoDBRepo.UpdateUserField(user.Username, "progress_level", user.Progress_Level+1)
    if err != nil {
        log.Printf("Error updating user progress_level: %v", err)
        return
    }

    // Update the user's coins
    err = uh.dynamoDBRepo.UpdateUserField(user.Username, "coins", user.Coins+100)
    if err != nil {
        log.Printf("Error updating user coins: %v", err)
        return
    }

    sendResponse(uh.channel, replyTo, correlationID, "UpdateProgressResponse", struct {
        Progress_Level int `json:"progress_level"`
        Coins int `json:"coins"`
    }{
        Progress_Level: user.Progress_Level + 1,
        Coins: user.Coins + 100,
    })
}

// Get the leaderboard for a user's country
func (uh *UserService) HandleGetCountryLeaderboard(data []byte, replyTo string, correlationID string) {
	var requestData struct {
		Action   string `json:"action"`
		Username string `json:"username"`
	}

	err := json.Unmarshal(data, &requestData)
	if err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		return
	}

    // Get the user's country
	users, err := uh.dynamoDBRepo.GetCountryLeaderboard(requestData.Username)
	if err != nil {
		log.Printf("Error fetching country leaderboard: %v", err)
		return
	}

	sendResponse(uh.channel, replyTo, correlationID, "GetCountryLeaderboardResponse", struct {
        Users []models.User `json:"users"`
    }{
        Users: users,
    })
}

// Get the global leaderboard
func (uh *UserService) HandleGetGlobalLeaderboard(data []byte, replyTo string, correlationID string) {
	var requestData struct {
		Action string `json:"action"`
	}

	err := json.Unmarshal(data, &requestData)
	if err != nil {
		log.Printf("Failed to unmarshal data: %v", err)
		return
	}

    // Get the global leaderboard
	users, err := uh.dynamoDBRepo.GetGlobalLeaderboard()
	if err != nil {
		log.Printf("Error fetching global leaderboard: %v", err)
		return
	}

	sendResponse(uh.channel, replyTo, correlationID, "GetGlobalLeaderboardResponse", struct {
		Users []models.User `json:"users"`
	}{
		Users: users,
	})
}
