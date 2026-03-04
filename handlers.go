package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Hit upload handler")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		http.Error(w, "Server configuration error", http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	agentName := r.FormValue("agent_name")
	tdUrl := r.FormValue("td_url")

	var text string
	var filename string

	// If audio is a URL (TrackDrive webhook flow)
	audioUrl := r.FormValue("audio")
	if audioUrl != "" {
		fmt.Printf("Received audio URL: %s\n", audioUrl)
		tdAuthHeader := os.Getenv("TD_AUTH_HEADER")
		if tdAuthHeader == "" {
			http.Error(w, "Server configuration error: missing TD_AUTH_HEADER", http.StatusInternalServerError)
			return
		}
		text, err = downloadAndTranscribeAudio(audioUrl, apiKey, tdAuthHeader)
		if err != nil {
			fmt.Printf("Error downloading/transcribing audio: %v\n", err)
			http.Error(w, "Failed to process audio URL", http.StatusInternalServerError)
			return
		}
		filename = audioUrl
	} else {
		// Fall back to direct file upload (batch script)
		file, header, fileErr := r.FormFile("audio")
		if fileErr != nil {
			fmt.Printf("Error retrieving file: %v\n", fileErr)
			http.Error(w, "Failed to retrieve audio", http.StatusBadRequest)
			return
		}
		defer file.Close()
		file.Seek(0, 0)

		fmt.Printf("File Size: %+v\n", header.Size)
		text, err = transcribeAudio(file, header.Filename, apiKey)
		if err != nil {
			fmt.Printf("Error transcribing audio: %v\n", err)
			http.Error(w, "Failed to transcribe", http.StatusInternalServerError)
			return
		}
		filename = header.Filename
	}

	fmt.Printf("Transcript: %s\n", text)

	callAnalysis, err := analyzeTranscript(text, apiKey)
	if err != nil {
		fmt.Printf("Error analyzing transcription: %v\n", err)
		http.Error(w, "Failed to analyze transcription", http.StatusInternalServerError)
		return
	}

	callAnalysis.Filename = filename
	callAnalysis.AgentName = agentName
	callAnalysis.TrackdriveUrl = tdUrl

	callId, err := SaveCall(callAnalysis)
	if err != nil {
		http.Error(w, "Failed to save call", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UploadResponse{ID: callId})
}

func getCallsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query params (default: limit=20, offset=0)
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20 // default
	offset := 0 // default

	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	if offsetStr != "" {
		fmt.Sscanf(offsetStr, "%d", &offset)
	}

	// Validate limits
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	// Get calls from DB
	calls, total, err := GetCalls(limit, offset)
	if err != nil {
		log.Printf("Error fetching calls: %v", err)
		http.Error(w, "Failed to fetch calls", http.StatusInternalServerError)
		return
	}

	// Build response
	response := CallsListResponse{
		Calls:  calls,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getCallByIdHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	call, err := GetCall(id)
	if err == sql.ErrNoRows {
		http.Error(w, "Call not found", http.StatusNotFound)
		return
	}

	if err != nil { // ← Any OTHER error
		log.Printf("Error fetching call: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(call)
}
