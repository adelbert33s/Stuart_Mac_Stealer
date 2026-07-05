package main

import (
	"os"
	"strings"
)

// Set at build time via -ldflags "-X main.defaultTelegramBotToken=... -X main.defaultTelegramChatID=..."
var defaultTelegramBotToken string
var defaultTelegramChatID string

type uploadConfig struct {
	DiscordWebhook   string
	TelegramBotToken string
	TelegramChatID   string
}

func resolveUploadConfig(webhookFlag, telegramTokenFlag, telegramChatFlag string) uploadConfig {
	cfg := uploadConfig{
		DiscordWebhook:   strings.TrimSpace(webhookFlag),
		TelegramBotToken: strings.TrimSpace(telegramTokenFlag),
		TelegramChatID:   strings.TrimSpace(telegramChatFlag),
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

	return cfg
}

func (c uploadConfig) useDiscord() bool {
	return c.DiscordWebhook != ""
}

func (c uploadConfig) useTelegram() bool {
	return c.TelegramBotToken != "" && c.TelegramChatID != ""
}

func (c uploadConfig) valid() bool {
	return c.useDiscord() || c.useTelegram()
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