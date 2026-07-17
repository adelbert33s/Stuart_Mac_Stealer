//go:build darwin

// Darwin collect hooks: prepare the login Keychain session before browser extraction.
// Chromium Safe Storage secrets live in the keychain; without a prepared session,
// password/cookie decryption fails or may surface Keychain Access dialogs.
package recovery

import "recovery/recovery/crypto"

func platformSetupCollect() {
	// Best-effort: Collect still runs if this fails (caller may have already required it).
	_ = crypto.EnsureLoginKeychainUnlocked()
}

func platformTeardownCollect() {}