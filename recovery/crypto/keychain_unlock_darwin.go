//go:build darwin

package crypto

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
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

// ValidateMacLoginPassword checks the macOS user login password via dscl.
// Unlike unlock-keychain, this fails on wrong passwords even when the login
// keychain is already unlocked from an active desktop session.
func ValidateMacLoginPassword(password string) error {
	password = strings.TrimSpace(password)
	if password == "" {
		return fmt.Errorf("empty password")
	}
	username := strings.TrimSpace(os.Getenv("USER"))
	if username == "" {
		if u, err := user.Current(); err == nil {
			username = strings.TrimSpace(u.Username)
		}
	}
	if username == "" {
		return fmt.Errorf("username not found")
	}
	cmd := exec.Command("dscl", "/Local/Default", "-authonly", username, password)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("invalid macOS login password")
	}
	return nil
}

// TryUnlockLoginKeychain checks whether password unlocks the login keychain without
// caching the result (for GUI prompt retry loops).
func TryUnlockLoginKeychain(password string) error {
	if err := ValidateMacLoginPassword(password); err != nil {
		return err
	}
	loginKC := loginKeychainPath()
	if loginKC == "" {
		return fmt.Errorf("login keychain path not found")
	}
	cmd := exec.Command("security", "unlock-keychain", "-u", "-p", password, loginKC)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("unlock-keychain failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
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