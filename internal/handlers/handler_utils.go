package handlers

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

func publishToRabbitMQ(ch *amqp.Channel, queueName, action string, data interface{}, replyTo string, correlationID string) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Failed to encode data to JSON: %v", err)
	}

	err = ch.Publish(
		"",
		queueName,
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
