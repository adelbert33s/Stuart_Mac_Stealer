package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const panelUploadTimeout = 120 * time.Second

func sendPanelUpload(panelURL, apiKey, title, summary, filename string, zipData []byte, meta panelUploadMeta) error {
	panelURL = strings.TrimSpace(panelURL)
	if panelURL == "" {
		return fmt.Errorf("panel URL is required")
	}
	if len(zipData) == 0 {
		return fmt.Errorf("empty upload archive")
	}

	endpoint := strings.TrimRight(panelURL, "/") + "/api/ingest"

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	writeField := func(name, value string) error {
		if value == "" {
			return nil
		}
		return w.WriteField(name, value)
	}

	if err := writeField("title", title); err != nil {
		return err
	}
	if err := writeField("summary", summary); err != nil {
		return err
	}
	if err := writeField("filename", filename); err != nil {
		return err
	}
	if err := writeField("hostname", meta.Hostname); err != nil {
		return err
	}
	if err := writeField("os", meta.OS); err != nil {
		return err
	}
	if err := writeField("arch", meta.Arch); err != nil {
		return err
	}
	if err := writeField("phase", meta.Phase); err != nil {
		return err
	}
	if meta.PartNum > 0 {
		if err := w.WriteField("part_num", strconv.Itoa(meta.PartNum)); err != nil {
			return err
		}
	}
	if meta.PartTotal > 0 {
		if err := w.WriteField("part_total", strconv.Itoa(meta.PartTotal)); err != nil {
			return err
		}
	}

	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(zipData); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	client := &http.Client{Timeout: panelUploadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("panel ingest HTTP %d: %s", resp.StatusCode, string(msg))
	}
	return nil
}

type panelUploadMeta struct {
	Hostname  string
	OS        string
	Arch      string
	Phase     string
	PartNum   int
	PartTotal int
}