//go:build darwin

// lockedfile_stub.go — read browser DBs that may be open by Chrome/Firefox.
//
// On Windows, ReadLockedFile can pull bytes via process handles. On macOS we
// simply os.ReadFile; Chromium often allows concurrent readers, and db.OpenDatabase
// already tries SQLite nolock first.
package platform

import "os"

// ReadLockedFile returns file contents; pids are ignored on darwin.
func ReadLockedFile(srcPath string, pids []uint32) ([]byte, error) {
	return os.ReadFile(srcPath)
}

// ResetHandleCache clears Windows-only handle caches (no-op on macOS).
func ResetHandleCache() {}
