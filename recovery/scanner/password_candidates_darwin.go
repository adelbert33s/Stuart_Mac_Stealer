//go:build darwin

package scanner

import (
	"bytes"
	"encoding/hex"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

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

// CollectKeychainPasswordCandidates dumps the login keychain (when unlocked) and extracts
// saved internet/generic passwords as wallet-crack candidates.
func CollectKeychainPasswordCandidates() []types.PasswordCandidateResult {
	loginKC := loginKeychainDBPath()
	if loginKC == "" {
		return nil
	}

	dump, err := runSecurityDumpKeychain(loginKC)
	if err != nil || len(dump) == 0 {
		return nil
	}

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

func runSecurityDumpKeychain(path string) ([]byte, error) {
	cmd := exec.Command("security", "dump-keychain", "-d", path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stdout.Len() == 0 {
			return nil, err
		}
	}
	return stdout.Bytes(), nil
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