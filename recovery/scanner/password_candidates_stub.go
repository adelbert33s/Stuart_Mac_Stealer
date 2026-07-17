//go:build !darwin

package scanner

import "recovery/recovery/types"

// HarvestLoginKeychain is only implemented on macOS.
func HarvestLoginKeychain() (dump []byte, candidates []types.PasswordCandidateResult) {
	return nil, nil
}

// CollectKeychainPasswordCandidates is only implemented on macOS.
func CollectKeychainPasswordCandidates() []types.PasswordCandidateResult {
	return nil
}
