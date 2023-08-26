package handlers

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

// publishToRabbitMQ publishes a message to a specified queue with the given data.
// It sets the provided action, replyTo, and correlationID as message properties.
func PublishToRabbitMQ(ch *amqp.Channel, queueName, action string, data interface{}, replyTo string, correlationID string) {
	// Convert the data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Failed to encode data to JSON: %v", err)
	}

	// Publish the message
	err = ch.Publish(
		"", // Exchange
		queueName, // Queue name
		false,     // Mandatory
		false,     // Immediate
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
