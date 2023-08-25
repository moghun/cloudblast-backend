package services

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

// sendResponse sends a response back to the caller
func sendResponse(ch *amqp.Channel, replyTo string, correlationID string, action string, data interface{}) {
    responseData := struct {
        Action string      `json:"action"`
        Data   interface{} `json:"data"`
    }{
        Action: action,
        Data:   data,
    }

    responseDataJSON, err := json.Marshal(responseData)
    if err != nil {
        log.Fatalf("Failed to encode response data to JSON: %v", err)
        return
    }

    // Publish response to the replyTo queue
    err = ch.Publish(
        "",
        replyTo,
        false,
        false,
        amqp.Publishing{
            ContentType:   "application/json",
            CorrelationId: correlationID,
            Body:          responseDataJSON,
            Headers: amqp.Table{
                "action": action,
            },
        },
    )
    if err != nil {
        log.Fatalf("Failed to publish a response message: %v", err)
    }
}