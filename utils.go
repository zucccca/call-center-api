package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/sashabaranov/go-openai"
)

type CallCompliance struct {
	Transcript      string   `json:"transcript"`
	Flags           []string `json:"flags"`
	FlagCount       int      `json:"flag_count"`
	IsPushy         bool     `json:"is_pushy"`
	Score           int      `json:"score"`
	Filename        string
	AgentName       string `json:"agent_name"`
	TrackdriveUrl   string `json:"trackdrive_url"`
	Disposition     string `json:"disposition"`
	OfferName       string `json:"offer_name"`
	AgentTalkTime   int    `json:"agent_talk_time"`
	ForwardDuration int    `json:"forward_duration"`
}

type CallSummary struct {
	ID              int       `json:"id"`
	Filename        string    `json:"filename"`
	Score           int       `json:"score"`
	FlagCount       int       `json:"flag_count"`
	IsPushy         bool      `json:"is_pushy"`
	CreatedAt       time.Time `json:"created_at"`
	AgentName       string    `json:"agent_name"`
	TrackdriveUrl   string    `json:"trackdrive_url"`
	Disposition     string    `json:"disposition"`
	OfferName       string    `json:"offer_name"`
	AgentTalkTime   int       `json:"agent_talk_time"`
	ForwardDuration int       `json:"forward_duration"`
}

type CallsListResponse struct {
	Calls  []CallSummary `json:"calls"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type CallDetail struct {
	ID              int       `json:"id"`
	Filename        string    `json:"filename"`
	Transcript      string    `json:"transcript"`
	Flags           []string  `json:"flags"`
	FlagCount       int       `json:"flag_count"`
	IsPushy         bool      `json:"is_pushy"`
	Score           int       `json:"score"`
	CreatedAt       time.Time `json:"created_at"`
	AgentName       string    `json:"agent_name"`
	TrackdriveUrl   string    `json:"trackdrive_url"`
	Disposition     string    `json:"disposition"`
	OfferName       string    `json:"offer_name"`
	AgentTalkTime   int       `json:"agent_talk_time"`
	ForwardDuration int       `json:"forward_duration"`
}

type OpenAIResponse struct {
	Text string `json:"text"`
}

type UploadResponse struct {
	ID int `json:"id"`
}

var systemPrompt = `
You are a strict Medicare Sales Compliance Officer. Analyze the provided call transcript and detect violations based on the categories below.

---

HARD VIOLATIONS — Phrases that must NEVER be said:

Guaranteed/Absolute Language:
- "I guarantee", "You will get", "You qualify", "You're approved", "You're eligible for sure"
- "There's no way you wouldn't qualify", "You won't pay anything", "This costs you nothing"
- "It's completely free", "Zero cost no matter what", "You'll definitely save money"
- "This will lower your premiums", "You can't lose", "This is the best plan"

Money/Cash Misrepresentation:
- "We can get you extra money", "Free money", "Cash back", "Stimulus benefit"
- "Government check", "Spending card with guaranteed balance", "Money added to your Social Security"
- "You're entitled to money", "We're giving out money"

Government/Medicare Affiliation:
- "I'm calling from Medicare", "I work with Medicare", "I'm your Medicare Advocate"
- "I'm with Social Security", "This is a Medicare program", "Medicare sent me"
- "We're partnered with Medicare", "This is a federal benefit", "This is required by Medicare"

Pressure/Coercion Language:
- "You need to do this", "You have to do this", "If you don't, you'll lose benefits"
- "This is your last chance", "You must act now", "You don't want to miss out"
- "I'll just sign you up", "Let's just get this done"

Enrollment Misrepresentation:
- "I'm enrolling you", "You're now signed up", "I've switched your plan"
- "I already updated your coverage", "We changed that for you"

---

BEHAVIORAL PATTERNS TO FLAG:

- Silent objections: Agent ignores customer objection or returns to script without addressing it
- Optional framing: Repeated use of "just checking", "just touching base", "if you want", "if you'd like", "up to you", "you can call back"
- Rushing the call: Rapid stacking of verifications, no engagement between questions, early transfer mention
- Rebuttal with no follow-up question: Agent delivers rebuttal then goes silent instead of asking a question
- Early transfer mention: Mentioning adviser or licensed agent before explaining value
- Overpromising benefits: Repeated emphasis on "extra benefits", leading with spending card, implied savings before review
- No expectation setting before transfer: Not explaining next steps or cold transferring
- Not framing as plan review: No mention of "Medicare Advantage plan options", positioning call as help instead of a review
- Tone issues: Hesitant, uncertain, low energy, or overly aggressive delivery

---

SCORING:
- Start at 100
- Deduct 15 points per hard violation phrase
- Deduct 10 points per behavioral pattern flagged
- Minimum score is 0

Set "is_pushy" to true if pressure/coercion language OR rushing patterns are detected.

Each entry in "flags" should be a short description of the specific violation found (e.g. "Used 'I guarantee' — guaranteed language", "Silent objection — ignored customer concern").

Output valid JSON only. No markdown. No preamble.
Format:
{
  "flags": ["description of violation 1", "description of violation 2"],
  "flag_count": 2,
  "is_pushy": false,
  "score": 70
}
`

func analyzeTranscript(transcript string, apiKey string) (*CallCompliance, error) {
	client := openai.NewClient(apiKey)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: transcript,
				},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
		},
	)

	if err != nil {
		return nil, err
	}

	var complianceData CallCompliance
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &complianceData)
	if err != nil {
		return nil, err
	}

	complianceData.Transcript = transcript
	return &complianceData, nil
}

func transcribeAudio(file multipart.File, filename string, apiKey string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}

	writer.WriteField("model", "whisper-1")
	err = writer.Close()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)
	var responseObj OpenAIResponse
	err = json.Unmarshal(responseBody, &responseObj)
	if err != nil {
		return "", err
	}

	return responseObj.Text, nil
}

func downloadAndTranscribeAudio(audioUrl string, apiKey string, tdAuthHeader string) (string, error) {
	// Download audio from TrackDrive URL
	req, err := http.NewRequest("GET", audioUrl, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("Authorization", tdAuthHeader)

	client := &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download audio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("audio download failed with status: %d", resp.StatusCode)
	}

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read audio bytes: %w", err)
	}

	// Pipe into Whisper
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return "", err
	}

	_, err = io.Copy(part, bytes.NewReader(audioBytes))
	if err != nil {
		return "", err
	}

	writer.WriteField("model", "whisper-1")
	err = writer.Close()
	if err != nil {
		return "", err
	}

	whisperReq, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", body)
	if err != nil {
		return "", err
	}

	whisperReq.Header.Set("Content-Type", writer.FormDataContentType())
	whisperReq.Header.Set("Authorization", "Bearer "+apiKey)

	whisperResp, err := client.Do(whisperReq)
	if err != nil {
		return "", err
	}
	defer whisperResp.Body.Close()

	responseBody, _ := io.ReadAll(whisperResp.Body)
	var responseObj OpenAIResponse
	err = json.Unmarshal(responseBody, &responseObj)
	if err != nil {
		return "", err
	}

	return responseObj.Text, nil
}
