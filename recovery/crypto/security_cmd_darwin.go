//go:build darwin

package crypto

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunSecurity runs security(1) with -mac-password and login keychain when configured.
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

// DumpLoginKeychain returns decrypted dump-keychain output when -mac-password is set.
func DumpLoginKeychain() ([]byte, error) {
	loginKC := loginKeychainPath()
	if loginKC == "" {
		return nil, fmt.Errorf("login keychain path not found")
	}
	out, err := RunSecurity("dump-keychain", "-d", loginKC)
	if err != nil && len(out) == 0 {
		return nil, err
	}
	return out, nil
}

func buildSecurityArgs(args ...string) []string {
	full := append([]string{}, args...)
	if macLoginPassword != "" {
		full = append(full, "-p", macLoginPassword)
	}
	loginKC := loginKeychainPath()
	if loginKC != "" && !securityArgsContain(full, loginKC) {
		full = append(full, loginKC)
	}
	return full
}

func configureSilentKeychainAccess(loginKC string) {
	if macLoginPassword == "" || loginKC == "" {
		return
	}

	// Keep login keychain unlocked; -u = do not lock when sleeping.
	_ = exec.Command("security", "set-keychain-settings", "-t", "3600", "-u", loginKC).Run()
	_ = exec.Command("security", "default-keychain", "-s", loginKC).Run()
	_ = exec.Command("security", "list-keychains", "-d", "-s", loginKC).Run()
	// No set-key-partition-list: wrong flags/order caused exit 2 + Keychain GUI prompts.
	// Silent reads use unlock-keychain + find-generic-password -p only.
}