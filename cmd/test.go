package main

import (
	"bytes"
	"encoding/json"
	"goodBlast-backend/internal/handlers"
	"goodBlast-backend/internal/services"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/streadway/amqp"
)

func TestPublishToRabbitMQ(t *testing.T) {
	ch := &amqp.Channel{}
	queueName := "testQueue"
	action := "sendMessage"
	data := struct {
		Message string `json:"message"`
	}{
		Message: "Hello, world!",
	}
	replyTo := "testReplyQueue"
	correlationID := "testCorrelationID"

	handlers.PublishToRabbitMQ(ch, queueName, action, data, replyTo, correlationID)
}

func TestHandleSearchUserRoute(t *testing.T) {
	// create a channel
	ch := amqp.Channel{}

	// create a request body
	reqBody := map[string]interface{}{
		"action":   "search",
		"username": "testuser",
	}
	jsonReq, _ := json.Marshal(reqBody)

	// create a request
	req, _ := http.NewRequest("POST", "/searchuser", bytes.NewReader(jsonReq))

	// create a response recorder
	recorder := httptest.NewRecorder()

	// invoke the handler
	handlers.HandleSearchUserRoute(&ch)(recorder, req)

	// check the response status code
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code 200, but got %d", recorder.Code)
	}

	// check the response body
	expectedRespBody := map[string]interface{}{
		"uid":                  "AAAAAAA",
		"username":             "testuser",
		"country":              "TR",
		"progress_level":       20,
		"coins":                100,
		"latest_tournament_id": "fasfdafa",
	}
	var actualRespBody map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &actualRespBody)
	if !reflect.DeepEqual(expectedRespBody, actualRespBody) {
		t.Errorf("Expected response body %v, but got %v", expectedRespBody, actualRespBody)
	}
}

func TestNewLeaderboardService(t *testing.T) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Errorf("Failed to connect to RabbitMQ: %v", err)
	}

	redisAddr := "localhost:6379"
	ls, err := services.NewLeaderboardService(conn, redisAddr)
	if err != nil {
		t.Errorf("Failed to create leaderboard service: %v", err)
	}

	// Check if leaderboard service is not nil
	if ls == nil {
		t.Error("Leaderboard service is nil")
	}

	defer conn.Close()
}

func TestNewUserService(t *testing.T) {
	conn := &amqp.Connection{}
	_, err := services.NewUserService(conn)
	if err != nil {
		t.Errorf("NewUserService error: %v", err)
	}
}

func TestHashPassword(t *testing.T) {
	password := "password"
	_, err := services.HashPassword(password)
	if err != nil {
		t.Errorf("hashPassword error: %v", err)
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "password"
	hashedPassword, err := services.HashPassword(password)
	if err != nil {
		t.Errorf("hashPassword error: %v", err)
	}

	result := services.CheckPasswordHash(password, hashedPassword)
	if !result {
		t.Errorf("checkPasswordHash returned false, expected true")
	}
}