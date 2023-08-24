package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

func publishToRabbitMQ(ch *amqp.Channel, action string, data interface{}, replyTo string, correlationID string) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Failed to encode data to JSON: %v", err)
	}

	err = ch.Publish(
		"",
		"userQueue",
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			ReplyTo:       replyTo,
			CorrelationId: correlationID,
			Body:          dataJSON,
			Headers: amqp.Table{
				"action": action,
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to publish a message: %v", err)
	}
}

func HandleSearchUserRoute(ch *amqp.Channel) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestData struct {
			Action   string `json:"action"`
			Username string `json:"username"`
		}

		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if requestData.Username == "" {
			http.Error(w, "Username is required", http.StatusBadRequest)
			return
		}

		correlationID := uuid.New().String()

		replyQueue, err := ch.QueueDeclare(
			"",
			false,
			true,
			true,
			false,
			nil,
		)
		if err != nil {
			http.Error(w, "Failed to create reply queue", http.StatusInternalServerError)
			return
		}

		msgs, err := ch.Consume(
			replyQueue.Name,
			"",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			http.Error(w, "Failed to set up reply consumer", http.StatusInternalServerError)
			return
		}

		publishToRabbitMQ(ch, "SearchUser", requestData, replyQueue.Name, correlationID)

		for msg := range msgs {
			if msg.CorrelationId == correlationID {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(msg.Body)
				return
			}
		}

		http.Error(w, "No response received", http.StatusRequestTimeout)
	}
}