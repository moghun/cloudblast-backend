package main

import (
	"fmt"
	"goodBlast-backend/internal/auth"
	"goodBlast-backend/internal/handlers"
	"goodBlast-backend/internal/services"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/streadway/amqp"
)

var conn *amqp.Connection
var ch *amqp.Channel


func main() {
	//Start RabbitMQ connection
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	//Start RabbitMQ channel
	ch, err = conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	//Declare RabbitMQ queue
	router := mux.NewRouter()
	router.HandleFunc("/api/tournament/StartTournament", handlers.HandleStartTournamentRoute(ch)).Methods("POST")
	router.HandleFunc("/api/tournament/EndTournament", handlers.HandleEndTournamentRoute(ch)).Methods("POST")
	router.HandleFunc("/api/user/CreateUser", handlers.HandleCreateUserRoute(ch)).Methods("POST")
	router.HandleFunc("/api/user/Login", handlers.HandleLoginRoute(ch)).Methods("GET")
	router.HandleFunc("/api/user/SearchUser", handlers.HandleSearchUserRoute(ch)).Methods("GET")
	router.HandleFunc("/api/user/UpdateProgress", auth.AuthMiddleware(handlers.HandleUpdateProgressRoute(ch))).Methods("POST")
	router.HandleFunc("/api/tournament/EnterTournament", auth.AuthMiddleware(handlers.HandleEnterTournamentRoute(ch))).Methods("POST")
	router.HandleFunc("/api/tournament/UpdateScore", auth.AuthMiddleware(handlers.HandleUpdateScoreRoute(ch))).Methods("POST")
	router.HandleFunc("/api/tournament/ClaimReward", auth.AuthMiddleware(handlers.HandleClaimRewardRoute(ch))).Methods("POST")
	router.HandleFunc("/api/tournament/GetTournamentRank", auth.AuthMiddleware(handlers.HandleGetGroupUserRankRoute(ch))).Methods("GET")
	router.HandleFunc("/api/tournament/GetTournamentLeaderboard", auth.AuthMiddleware(handlers.HandleGetGroupLeaderboardWithRanksRoute(ch))).Methods("GET")

	//Start user service
	userService, err := services.NewUserService(conn)

	if err != nil {
		log.Fatalf("Failed to initialize user_handler: %v", err)
	}
	go userService.Start()

	// Start tournament service
	tournamentService, err := services.NewTournamentService(conn)
	if err != nil {
		log.Fatalf("Failed to initialize tournament_service: %v", err)
	}
	go tournamentService.Start()

	leaderboardService, err := services.NewLeaderboardService(conn, "localhost:6379")
	if err != nil {
		log.Fatalf("Failed to initialize tournament_service: %v", err)
	}
	go leaderboardService.Start()

	//Handle graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	//Start HTTP server
	go func() {
		log.Println("Server is starting...")
		log.Fatal(http.ListenAndServe(":8080", router))
	}()
	<-stopChan
	fmt.Println("Main service is stopping...")
	userService.Stop()
	tournamentService.Stop()
	leaderboardService.Stop()
	fmt.Println("Main service stopped.")
}