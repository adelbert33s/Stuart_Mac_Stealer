//go:build darwin

package crypto

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

var (
	macLoginPassword  string
	keychainOnce      sync.Once
	keychainUnlocked  bool
	keychainUnlockErr error
)

// SetMacLoginPassword stores the macOS user/login password for silent keychain unlock.
func SetMacLoginPassword(password string) {
	macLoginPassword = strings.TrimSpace(password)
}

// MacLoginPassword returns the configured macOS login password, if any.
func MacLoginPassword() string {
	return macLoginPassword
}

// loginKeychainAlreadyUsable reports whether keychain reads work without unlock-keychain.
func loginKeychainAlreadyUsable() bool {
	loginKC := loginKeychainPath()
	if loginKC == "" {
		return false
	}
	cmd := exec.Command("security", "find-generic-password", "-s", "Chrome Safe Storage", "-a", "Chrome", "-w", loginKC)
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

// TryUnlockLoginKeychain validates the login password without calling lock/unlock-keychain.
func TryUnlockLoginKeychain(password string) error {
	return ValidateMacLoginPassword(password)
}

// EnsureLoginKeychainUnlocked unlocks login.keychain-db without a GUI prompt when the
// macOS login password was provided via -mac-password or KEMATIAN_MAC_PASSWORD.
func EnsureLoginKeychainUnlocked() error {
	keychainOnce.Do(func() {
		if macLoginPassword == "" {
			return
		}
		loginKC := loginKeychainPath()
		if loginKC == "" {
			keychainUnlockErr = fmt.Errorf("login keychain path not found")
			return
		}
		if loginKeychainAlreadyUsable() {
			keychainUnlocked = true
			configureSilentKeychainAccess(loginKC)
			logf("login keychain already usable (skipped unlock-keychain)")
			return
		}
		cmd := exec.Command("security", "unlock-keychain", "-u", "-p", macLoginPassword, loginKC)
		if out, err := cmd.CombinedOutput(); err != nil {
			keychainUnlockErr = fmt.Errorf("unlock-keychain failed: %w (%s)", err, strings.TrimSpace(string(out)))
			return
		}
		keychainUnlocked = true
		configureSilentKeychainAccess(loginKC)
		logf("login keychain unlocked via -mac-password")
	})
	return keychainUnlockErr
}

// LoginKeychainUnlocked reports whether silent unlock succeeded.
func LoginKeychainUnlocked() bool {
	return keychainUnlocked
}