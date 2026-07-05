//go:build darwin

package recovery

import "recovery/recovery/crypto"

func platformSetupCollect() {
	_ = crypto.EnsureLoginKeychainUnlocked()
}

func platformTeardownCollect() {}