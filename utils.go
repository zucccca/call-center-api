package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
)

type CallCompliance struct {
	Flags []string `json:"flags"`
  FlagCount int `json:"flag_count"`
  Transcript string `json:"transcript"`
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

	return string(responseBody), nil
}