package main

import (
	"fmt"
	"net/http"
	"os"
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
		fmt.Printf("Error retrieving file: %v\n", err) 
    http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
    return
	}

	defer file.Close()
	file.Seek(0,0) // go to start of file

	apiKey := os.Getenv("OPENAI_API_KEY")

	if apiKey == "" {
  	http.Error(w, "Server configuration error", http.StatusInternalServerError)
    return
	}

	text, err := transcribeAudio(file, header.Filename, apiKey)

	if err != nil {
		fmt.Printf("Error processing audio: %v\n", err)
		http.Error(w, "Failed to transcribe", http.StatusInternalServerError)
		return
	}

	fmt.Printf("OpenAI Response: %s\n", text)

	fmt.Printf("Uploaded FIle: %+v\n", header.Filename)
	fmt.Printf("File Size: %+v\n", header.Size)

	fmt.Fprintf(w, "Successfully Uploaded File\n")

}