//go:build darwin

package crypto

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

var (
	macLoginPassword string
	keychainOnce     sync.Once
	keychainUnlocked bool
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
		// -u updates unlock timeout; avoids repeated prompts during harvest.
		cmd := exec.Command("security", "unlock-keychain", "-u", "-p", macLoginPassword, loginKC)
		if out, err := cmd.CombinedOutput(); err != nil {
			keychainUnlockErr = fmt.Errorf("unlock-keychain failed: %w (%s)", err, strings.TrimSpace(string(out)))
			return
		}
		keychainUnlocked = true
		logf("login keychain unlocked via -mac-password")
	})
	return keychainUnlockErr
}

// LoginKeychainUnlocked reports whether silent unlock succeeded.
func LoginKeychainUnlocked() bool {
	return keychainUnlocked
}