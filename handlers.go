package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func getDispositionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query(`SELECT DISTINCT disposition FROM calls WHERE disposition != '' ORDER BY disposition`)
	if err != nil {
		log.Printf("Error fetching dispositions: %v", err)
		http.Error(w, "Failed to fetch dispositions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	dispositions := []string{}
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err == nil {
			dispositions = append(dispositions, d)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dispositions)
}

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

	var agentName, tdUrl, audioUrl, disposition, offerName string
	var agentTalkTime, forwardDuration int
	var text, filename string
	var err error

	contentType := r.Header.Get("Content-Type")
	fmt.Println("Content-Type:", contentType)

	if strings.Contains(contentType, "application/json") {
		var payload struct {
			Audio           string `json:"audio"`
			AgentName       string `json:"agent_name"`
			TdUrl           string `json:"td_url"`
			Disposition     string `json:"disposition"`
			OfferName       string `json:"offer_name"`
			AgentTalkTime   string `json:"agent_talk_time"`
			ForwardDuration string `json:"forward_duration"`
		}
		if err = json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
			return
		}
		audioUrl = payload.Audio
		agentName = payload.AgentName
		tdUrl = payload.TdUrl
		disposition = payload.Disposition
		offerName = payload.OfferName

		if f, err := strconv.ParseFloat(payload.AgentTalkTime, 64); err == nil {
			agentTalkTime = int(f)
		}
		if f, err := strconv.ParseFloat(payload.ForwardDuration, 64); err == nil {
			forwardDuration = int(f)
		}
	}

	if audioUrl != "" {
		// Audio is a URL — download and transcribe
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
	callAnalysis.Disposition = disposition
	callAnalysis.OfferName = offerName
	callAnalysis.AgentTalkTime = agentTalkTime
	callAnalysis.ForwardDuration = forwardDuration

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

	filters := CallFilters{
		AgentName:   r.URL.Query().Get("agent_name"),
		Disposition: r.URL.Query().Get("disposition"),
		SortBy:      r.URL.Query().Get("sort_by"),
		SortOrder:   r.URL.Query().Get("sort_order"),
		DateFrom:    r.URL.Query().Get("date_from"),
		DateTo:      r.URL.Query().Get("date_to"),
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	offset := 0

	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	if offsetStr != "" {
		fmt.Sscanf(offsetStr, "%d", &offset)
	}

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	calls, total, err := GetCalls(limit, offset, filters)
	if err != nil {
		log.Printf("Error fetching calls: %v", err)
		http.Error(w, "Failed to fetch calls", http.StatusInternalServerError)
		return
	}

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

	if err != nil {
		log.Printf("Error fetching call: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(call)
}
