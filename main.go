package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Hit upload handler")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10 << 20) // limit request to 10mb
	err := r.ParseMultipartForm( 10 << 20)

	if err != nil {
		http.Error(w, "File too big", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("audio")

	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}

	defer file.Close()

	fmt.Printf("Uploaded FIle: %+v\n", header.Filename)
	fmt.Printf("File Size: %+v\n", header.Size)

	fmt.Fprintf(w, "Successfully Uploaded File\n")

}

func main() {
	err := godotenv.Load()
	apiKey := os.Getenv("OPENAI_API_KEY")

	if err != nil {
		log.Println("No .env file found, relying on system env")
	}

	if apiKey == "" {
		log.Fatal("Error: OPEN_AI_API_KEY is missing")
	}

	// openai := openai.NewClient(apiKey)
	r := chi.NewRouter()
	port := ":4000"
	
	r.HandleFunc("/upload", uploadHandler)
	log.Printf("Server listening on port %s", port)
	
	listenErr := http.ListenAndServe(port, r)

	if listenErr != nil {
		log.Fatalf("Server failed to listen on port %s", port)
	}

}