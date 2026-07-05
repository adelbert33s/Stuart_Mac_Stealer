package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"recovery/recovery/browser"
	"recovery/recovery/types"
)

// ScanExtensions returns installed Chromium crypto wallet extensions only.
func ScanExtensions() []types.ExtensionResult {
	var results []types.ExtensionResult
	for _, cfg := range browser.Browsers {
		if cfg.IsFirefox {
			continue
		}
		profiles := browser.FindProfileDirs(cfg)
		for _, profile := range profiles {
			extDir := filepath.Join(profile.Path, "Extensions")
			entries, err := os.ReadDir(extDir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				extID := e.Name()
				// Skip internal Chromium marker dirs
				if strings.HasPrefix(extID, "_") {
					continue
				}
				extIDDir := filepath.Join(extDir, extID)
				versionDirs, err := os.ReadDir(extIDDir)
				if err != nil {
					continue
				}
				walletName, isWallet := knownWalletExtensions[extID]
				if !isWallet {
					continue
				}
				for _, vd := range versionDirs {
					if !vd.IsDir() {
						continue
					}
					versionPath := filepath.Join(extIDDir, vd.Name())
					name, version := readManifestBasics(filepath.Join(versionPath, "manifest.json"))
					if name == "" || strings.HasPrefix(name, "__MSG_") {
						if walletName != "" && !strings.HasPrefix(walletName, "Extension ") {
							name = walletName
						}
					}
					if name == "" || strings.HasPrefix(name, "__MSG_") {
						name = extID
					}
					results = append(results, types.ExtensionResult{
						ExtID:    extID,
						Name:     name,
						Version:  version,
						Browser:  cfg.Name,
						Profile:  profile.Name,
						Path:     versionPath,
						Category: "wallet",
					})
					break // first version directory only
				}
			}
		}
	}
	return results
}

type manifestBasics struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func readManifestBasics(path string) (name, version string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	var m manifestBasics
	if err := json.Unmarshal(data, &m); err != nil {
		return "", ""
	}
	if strings.HasPrefix(m.Name, "__MSG_") {
		m.Name = ""
	}
	return m.Name, m.Version
}
