package handlers

import (
	"encoding/json"
	"goodBlast-backend/internal/models"
	"net/http"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

func HandleStartTournamentRoute(ch *amqp.Channel) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Action string `json:"action"`
        }

        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
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

        publishToRabbitMQ(ch, "tournamentQueue", "StartTournament", requestData, replyQueue.Name, correlationID)

        for msg := range msgs {
            if msg.CorrelationId == correlationID {
                var response struct {
                    Action  string                 `json:"action"`
                    Data    map[string]interface{} `json:"data"`
                }

                err := json.Unmarshal(msg.Body, &response)
                if err != nil {
                    http.Error(w, "Failed to unmarshal response data", http.StatusInternalServerError)
                    return
                }

                data := response.Data

                errorVal, errorExists := data["error"]
                if errorExists {
                    http.Error(w, errorVal.(string), http.StatusInternalServerError)
                    return
                }

                tournamentID := data["tournament_id"].(string)
                startTime := data["start_time"].(string)
                endTime := data["end_time"].(string)

                responseData := struct {
                    TournamentID string `json:"tournament_id"`
                    StartTime    string `json:"start_time"`
                    EndTime      string `json:"end_time"`
					NumRegisteredUsers int `json:"num_registered_users"`
					Finished 	 bool `json:"finished"`
                }{
                    TournamentID: tournamentID,
                    StartTime:    startTime,
                    EndTime:      endTime,
					NumRegisteredUsers: 0,
					Finished: false,
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

func HandleEnterTournamentRoute(ch *amqp.Channel) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Action       string `json:"action"`
            Username     string `json:"username"`
            TournamentID string `json:"tournament_id"`
        }

        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
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

        publishToRabbitMQ(ch, "tournamentQueue", "EnterTournament", requestData, replyQueue.Name, correlationID)

        for msg := range msgs {
            if msg.CorrelationId == correlationID {
                var response struct {
                    Action  string                 `json:"action"`
                    Data    map[string]interface{} `json:"data"`
                }

                err := json.Unmarshal(msg.Body, &response)
                if err != nil {
                    http.Error(w, "Failed to unmarshal response data", http.StatusInternalServerError)
                    return
                }

                data := response.Data

                errorVal, errorExists := data["error"]
                if errorExists {
                    http.Error(w, errorVal.(string), http.StatusInternalServerError)
                    return
                }

                groupID := int(data["group_id"].(float64))

                responseData := struct {
                    GroupID int `json:"group_id"`
                }{
                    GroupID: groupID,
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

func HandleUpdateScoreRoute(ch *amqp.Channel) func(http.ResponseWriter, *http.Request) {
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

        publishToRabbitMQ(ch, "tournamentQueue", "UpdateScore", requestData, replyQueue.Name, correlationID)

        for msg := range msgs {
            if msg.CorrelationId == correlationID {
                var response struct {
                    Action  string                 `json:"action"`
                    Data    map[string]interface{} `json:"data"`
                }

                err := json.Unmarshal(msg.Body, &response)
                if err != nil {
                    http.Error(w, "Failed to unmarshal response data", http.StatusInternalServerError)
                    return
                }

                data := response.Data

                errorVal, errorExists := data["error"]
                if errorExists {
                    http.Error(w, errorVal.(string), http.StatusInternalServerError)
                    return
                }

                progressLevel := int(data["progress_level"].(float64))
                coins := int(data["coins"].(float64))
                score := int(data["score"].(float64))

                responseData := struct {
                    Progress_Level int `json:"progress_level"`
                    Coins         int `json:"coins"`
                    Score         int `json:"score"`
                }{
                    Progress_Level: progressLevel,
                    Coins:         coins,
                    Score:         score,
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

func HandleEndTournamentRoute(ch *amqp.Channel) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Action string `json:"action"`
        }

        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
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

        publishToRabbitMQ(ch, "tournamentQueue", "EndTournament", requestData, replyQueue.Name, correlationID)

        for msg := range msgs {
            if msg.CorrelationId == correlationID {
                var response struct {
                    Action  string                      `json:"action"`
                    Data    map[string][]models.UserInTournament `json:"data"`
                }

                err := json.Unmarshal(msg.Body, &response)
                if err != nil {
                    http.Error(w, "Failed to unmarshal response data", http.StatusInternalServerError)
                    return
                }

                data := response.Data

                rankedPlayers := data["ranked_players"]

                responseData := struct {
                    RankedPlayers []models.UserInTournament `json:"ranked_players"`
                }{
                    RankedPlayers: rankedPlayers,
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

func HandleClaimRewardRoute(ch *amqp.Channel) func(http.ResponseWriter, *http.Request) {
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

        publishToRabbitMQ(ch, "tournamentQueue", "ClaimReward", requestData, replyQueue.Name, correlationID)

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
                success, successExists := data["success"].(bool)
                reward_claimed, reward_claimedExist := data["reward_claimed"].(float64)
                errorVal, errorExists := data["error"].(string)

                if errorExists {
                    http.Error(w, errorVal, http.StatusInternalServerError)
                    return
                }

                if (!successExists || !reward_claimedExist) && !errorExists {
                    http.Error(w, "success or reward_claimed field not found in response data", http.StatusInternalServerError)
                    return
                }
                responseData := struct {
                    Success       bool `json:"success"`
                    RewardClaimed int  `json:"reward_claimed"`
                }{
                    Success:       success,
                    RewardClaimed: int(reward_claimed),
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

func HandleGetGroupUserRankRoute(ch *amqp.Channel) func(http.ResponseWriter, *http.Request) {
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

        publishToRabbitMQ(ch, "leaderboardQueue", "GetGroupUserRank", requestData, replyQueue.Name, correlationID)

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
                rank, rankExists := data["rank"].(float64)
                errorVal, errorExists := data["error"].(string)

                if errorExists {
                    http.Error(w, errorVal, http.StatusInternalServerError)
                    return
                }

                if !rankExists && !errorExists {
                    http.Error(w, "rank field not found in response data", http.StatusInternalServerError)
                    return
                }
                responseData := struct {
                    Rank int `json:"rank"`
                }{
                    Rank: int(rank),
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

func HandleGetGroupLeaderboardWithRanksRoute(ch *amqp.Channel) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Action          string `json:"action"`
            Username string `json:"username"`
        }

        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
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

        publishToRabbitMQ(ch, "leaderboardQueue", "GetGroupLeaderboardWithRanks", requestData, replyQueue.Name, correlationID)

        for msg := range msgs {
            if msg.CorrelationId == correlationID {
                var response struct {
                    Action string                 `json:"action"`
                    Data   []map[string]interface{} `json:"data"`
                }

                err := json.Unmarshal(msg.Body, &response)
                if err != nil {
                    http.Error(w, "Failed to unmarshal response data", http.StatusInternalServerError)
                    return
                }
                data := response.Data
                
                responseData := struct {
                    Data []map[string]interface{} `json:"data"`
                }{
                    Data: data,
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
