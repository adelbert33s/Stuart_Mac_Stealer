//go:build darwin

package crypto

import (
	"fmt"
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

// TryUnlockLoginKeychain validates the login password without touching the keychain.
func TryUnlockLoginKeychain(password string) error {
	return ValidateMacLoginPassword(password)
}

// EnsureLoginKeychainUnlocked prepares the login keychain for reads in the current GUI session.
// It intentionally does NOT call unlock-keychain: that command opens the real macOS Keychain
// password dialog from unsigned binaries even when -p is supplied.
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
		keychainUnlocked = true
		configureSilentKeychainAccess(loginKC)
		logf("keychain session ready (unlock-keychain disabled — uses logged-in session)")
	})
	return keychainUnlockErr
}

// LoginKeychainUnlocked reports whether silent unlock succeeded.
func LoginKeychainUnlocked() bool {
	return keychainUnlocked
}