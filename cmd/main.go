package main

import (
	"fmt"
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
	router.HandleFunc("/api/CreateUser", handlers.HandleCreateUserRoute(ch)).Methods("POST")
	router.HandleFunc("/api/Login", handlers.HandleLoginRoute(ch)).Methods("POST")
	router.HandleFunc("/api/SearchUser", handlers.HandleSearchUserRoute(ch)).Methods("POST")
	router.HandleFunc("/api/UpdateProgress", handlers.HandleUpdateProgressRoute(ch)).Methods("POST")



	//Start user service
	userService, err := services.NewUserService(conn)
	if err != nil {
		log.Fatalf("Failed to initialize user_handler: %v", err)
	}
	go userService.Start()

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
	fmt.Println("Main service stopped.")
}