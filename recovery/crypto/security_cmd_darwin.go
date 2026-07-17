//go:build darwin

// security_cmd_darwin.go — wrappers around /usr/bin/security for keychain I/O.
//
// unlock-keychain is handled only in keychain_unlock_darwin.go (with -p).
// find-generic-password / dump-keychain run against an already-unlocked session.
package crypto

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunSecurity runs security(1) against the login keychain after ensuring unlock.
func RunSecurity(args ...string) ([]byte, error) {
	_ = EnsureLoginKeychainUnlocked()
	cmd := exec.Command("security", buildSecurityArgs(args...)...)
	return cmd.CombinedOutput()
}

// RunSecurityStdout runs security(1) and returns trimmed stdout (for -w lookups).
func RunSecurityStdout(args ...string) (string, error) {
	_ = EnsureLoginKeychainUnlocked()
	cmd := exec.Command("security", buildSecurityArgs(args...)...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("%w: %s", err, msg)
		}
		return "", err
	}
	if strings.Contains(stderr.String(), "could not be found") {
		return "", fmt.Errorf("could not be found")
	}
	return strings.TrimSpace(stdout.String()), nil
}

// DumpLoginKeychain returns decrypted dump-keychain -d output for the login keychain.
// Requires EnsureLoginKeychainUnlocked (unlock + set-key-partition-list) first so
// dump does not pop system Allow dialogs for each item.
func DumpLoginKeychain() ([]byte, error) {
	loginKC := loginKeychainPath()
	if loginKC == "" {
		return nil, fmt.Errorf("login keychain path not found")
	}
	if err := EnsureLoginKeychainUnlocked(); err != nil {
		return nil, err
	}
	// dump-keychain -d: include plaintext passwords (keychain unlocked + partition list).
	cmd := exec.Command("security", "dump-keychain", "-d", loginKC)
	out, err := cmd.CombinedOutput()
	if len(out) == 0 && err != nil {
		return nil, fmt.Errorf("dump-keychain: %w", err)
	}
	// Partial dumps can still be useful when some items deny access.
	return out, nil
}

func buildSecurityArgs(args ...string) []string {
	full := append([]string{}, args...)
	// Append login keychain path when not already present so lookups hit the right DB.
	// Never inject -p here — only unlock-keychain accepts it.
	loginKC := loginKeychainPath()
	if loginKC != "" && !securityArgsContain(full, loginKC) {
		full = append(full, loginKC)
	}
	return full
}

func configureSilentKeychainAccess(loginKC string) {
	if loginKC == "" {
		return
	}

	// Keep login keychain unlocked longer; -u = do not lock when sleeping.
	// Do not replace the full keychain search list (that would drop System.keychain).
	_ = exec.Command("security", "set-keychain-settings", "-t", "3600", "-u", loginKC).Run()
	_ = exec.Command("security", "default-keychain", "-s", loginKC).Run()
}