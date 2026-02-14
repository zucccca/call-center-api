package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)



func main() {
	
	err := godotenv.Load()
	apiKey := os.Getenv("OPENAI_API_KEY")

	if err != nil {
		log.Println("No .env file found, relying on system env")
	}

	if apiKey == "" {
		log.Fatal("Error: OPEN_AI_API_KEY is missing")
	}

	connStr := os.Getenv("DATABASE_URL")

	if connStr == "" {
		log.Fatal("Error: DATABASE_URL is missing")
	}

	db, err = InitDB(connStr)

	if err != nil {
		log.Fatalf("Failed making connection to database %v", err)
	}

	r := chi.NewRouter()
	port := ":4000"
	
	r.HandleFunc("/upload", uploadHandler)
	r.HandleFunc("/calls", getCallsHandler)
	
	log.Printf("Server listening on port %s", port)
	
	listenErr := http.ListenAndServe(port, r)

	if listenErr != nil {
		log.Fatalf("Server failed to listen on port %s", port)
	}

}