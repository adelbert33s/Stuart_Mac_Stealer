//go:build darwin

package scanner

import (
	"os"
	"path/filepath"
)

func getDesktopWalletPaths() []walletConfig {
	return []walletConfig{
		{"Atomic", "atomic/Local Storage/leveldb", "appdata"},
		{"Exodus", "Exodus/exodus.wallet", "appdata"},
		{"Electrum", "Electrum/wallets", "home_dot"},
		{"Ethereum", "Ethereum/keystore", "home_dot"},
		{"Coinomi", "Coinomi/wallets", "appdata"},
	}
}

func resolveWalletBase(base string) string {
	home, _ := os.UserHomeDir()
	switch base {
	case "home", "userprofile", "home_dot":
		if base == "home_dot" {
			return filepath.Join(home, ".")
		}
		return home
	case "appdata", "localappdata":
		return filepath.Join(home, "Library", "Application Support")
	}
	return ""
}