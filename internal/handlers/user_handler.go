package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

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

		publishToRabbitMQ(ch, "userQueue", "SearchUser", requestData, replyQueue.Name, correlationID)

		for msg := range msgs {
			if msg.CorrelationId == correlationID {
				var response map[string]interface{}
				err := json.Unmarshal(msg.Body, &response)
				if err != nil {
					http.Error(w, "Failed to unmarshal response data", http.StatusInternalServerError)
					return
				}

				userData, userExists := response["user"]
				if !userExists {
					http.Error(w, "User data not found in response", http.StatusInternalServerError)
					return
				}

				userJSON, err := json.Marshal(userData)
				if err != nil {
					http.Error(w, "Failed to marshal user data", http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(userJSON)
				return
			}
		}

		http.Error(w, "No response received", http.StatusRequestTimeout)
	}
}

func HandleCreateUserRoute(ch *amqp.Channel) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Action   string `json:"action"`
            Username string `json:"username"`
            Password string `json:"password"`
        }

        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
            return
        }

        if requestData.Username == "" || requestData.Password == "" {
            http.Error(w, "Username and password are required", http.StatusBadRequest)
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

        publishToRabbitMQ(ch, "userQueue", "CreateUser", requestData, replyQueue.Name, correlationID)

        for msg := range msgs {
			if msg.CorrelationId == correlationID {
				var response struct {
					Action  string `json:"action"`
					Data    map[string]interface{} `json:"data"`
				}

				err := json.Unmarshal(msg.Body, &response)
				if err != nil {
					http.Error(w, "Failed to unmarshal response data", http.StatusInternalServerError)
					return
				}

				data := response.Data

				errorVal, errorExists := data["error"]; 
				if errorExists == true {
					http.Error(w, errorVal.(string), http.StatusInternalServerError)
					return
				}

                userID := data["user_id"].(string)

                responseData := struct {
                    UserID string `json:"user_id"`
                }{
                    UserID: userID,
                }

                responseDataJSON, err := json.Marshal(responseData)
                if err != nil {
                    http.Error(w, "Failed to marshal response data", http.StatusInternalServerError)
                    return
                }

                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                w.Write(responseDataJSON)
                return
            }
        }

        http.Error(w, "No response received", http.StatusRequestTimeout)
    }
}

func HandleLoginRoute(ch *amqp.Channel) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Action   string `json:"action"`
            Username string `json:"username"`
            Password string `json:"password"`
        }

        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
            return
        }

        if requestData.Username == "" || requestData.Password == "" {
            http.Error(w, "Username and password are required", http.StatusBadRequest)
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

        publishToRabbitMQ(ch, "userQueue", "Login", requestData, replyQueue.Name, correlationID)

        for msg := range msgs {
			if msg.CorrelationId == correlationID {
				var response struct {
					Action  string `json:"action"`
					Data    map[string]interface{} `json:"data"`
				}

				err := json.Unmarshal(msg.Body, &response)
				if err != nil {
					http.Error(w, "Failed to unmarshal response data", http.StatusInternalServerError)
					return
				}

				data := response.Data

				success, successExists := data["success"].(bool)
				if !successExists {
					http.Error(w, "Success field not found in response data", http.StatusInternalServerError)
					return
				}

				if success {
					token, tokenExists := data["token"].(string)
					if !tokenExists {
						http.Error(w, "Token not found in response data", http.StatusInternalServerError)
						return
					}

					responseData := struct {
						Token string `json:"jwt_token"`
					}{
						Token: token,
					}

					responseDataJSON, err := json.Marshal(responseData)
					if err != nil {
						http.Error(w, "Failed to marshal response data", http.StatusInternalServerError)
						return
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write(responseDataJSON)
					return
				} else {
					w.WriteHeader(http.StatusUnauthorized)
				}

				return
			}
		}

        http.Error(w, "No response received", http.StatusRequestTimeout)
    }
}

func HandleUpdateProgressRoute(ch *amqp.Channel) func(http.ResponseWriter, *http.Request) {
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

        publishToRabbitMQ(ch, "userQueue", "UpdateProgress", requestData, replyQueue.Name, correlationID)

        for msg := range msgs {
            if msg.CorrelationId == correlationID {
                var response struct {
                    Action string                 `json:"action"`
                    Data   map[string]interface{} `json:"data"`
                }

                err := json.Unmarshal(msg.Body, &response)
                if err != nil {
                    http.Error(w, "Failed to unmarshal response data", http.StatusInternalServerError)
                    return
                }

                data := response.Data

                progress_level, progress_levelExists := data["progress_level"].(float64)
                coins, coinsExists := data["coins"].(float64)
                if !progress_levelExists || !coinsExists {
                    http.Error(w, "Progress Level or coins field not found in response data", http.StatusInternalServerError)
                    return
                }

                responseData := struct {
                    Progress_Level int `json:"progress_level"`
                    Coins int `json:"coins"`
                }{
                    Progress_Level: int(progress_level),
                    Coins: int(coins),
                }

                responseDataJSON, err := json.Marshal(responseData)
                if err != nil {
                    http.Error(w, "Failed to marshal response data", http.StatusInternalServerError)
                    return
                }

                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                w.Write(responseDataJSON)
                return
            }
        }

        http.Error(w, "No response received", http.StatusRequestTimeout)
    }
}
