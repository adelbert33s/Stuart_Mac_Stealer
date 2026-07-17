//go:build !darwin

package crypto

// Non-darwin stubs — Keychain unlock/dump only exist on macOS.

func SetMacLoginPassword(password string) {}

func MacLoginPassword() string { return "" }

func TryUnlockLoginKeychain(password string) error {
	return nil
}

func EnsureLoginKeychainUnlocked() error { return nil }

func LoginKeychainUnlocked() bool { return false }

func LoginKeychainWasLocked() bool { return false }

func DumpLoginKeychain() ([]byte, error) { return nil, nil }

func IsLoginKeychainLocked(loginKC string) bool { return true }
