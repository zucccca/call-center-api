package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/sashabaranov/go-openai"
)

type CallCompliance struct {
	Transcript string `json:"transcript"`
	Flags []string `json:"flags"`
  FlagCount int `json:"flag_count"`
  IsPushy bool `json:"is_pushy"`
	Score int `json:"score"`
	Filename string
}

type OpenAIResponse struct {
    Text string `json:"text"`
}

type UploadResponse struct {
    ID int `json:"id"`
}

var systemPrompt = `
	You are a strict Medicare QA Compliance Officer. 
	Analyze the user's call transcript for Non-Compliant Activity (NCA).

	Your Rules:
	1. Detect Flags: Identify specific instances of prohibited words: "guarantee", "promise", "refund", "cancel", "credit card".
	2. Detect Steamrolling: Did the agent interrupt or ignore the customer? Set "is_pushy" to true.
	3. Score: Start at 100. Deduct 10 points for every flag. Deduct 20 points for steamrolling.

	Output valid JSON only. Do not include markdown formatting.
	Format:
	{
		"flags": ["guarantee", "refund"],
		"flag_count": 2,
		"is_pushy": true,
		"score": 70
	}
	`


func analyzeTranscript(transcript string, apiKey string) (*CallCompliance, error) {
	client := openai.NewClient(apiKey)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
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
		return  "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer " + apiKey)

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