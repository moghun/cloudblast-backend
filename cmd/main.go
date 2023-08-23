package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	//"goodBlast-backend/internal/handlers"
)

func main() {
    r := mux.NewRouter()

    http.Handle("/", r)

    log.Println("Server is starting...")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
