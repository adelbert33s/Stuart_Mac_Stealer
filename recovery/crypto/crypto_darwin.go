//go:build darwin

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"recovery/recovery/browser"
	"recovery/recovery/types"

	"golang.org/x/crypto/pbkdf2"
)

const (
	darwinChromeSalt       = "saltysalt"
	darwinChromeIterations = 1003
	darwinChromeKeyLen     = 16
)

var chromeKeychainServices = map[string]string{
	"Chrome":        "Chrome Safe Storage",
	"Chrome Beta":   "Chrome Safe Storage",
	"Chrome Canary": "Chrome Safe Storage",
	"Chromium":      "Chromium Safe Storage",
	"Edge":          "Microsoft Edge Safe Storage",
	"Brave":         "Brave Safe Storage",
	"Vivaldi":       "Vivaldi Safe Storage",
	"Opera":         "Opera Safe Storage",
	"Opera GX":      "Opera Safe Storage",
	"Arc":           "Arc Safe Storage",
	"Yandex":        "Yandex Safe Storage",
}

func chromeKeychainService(browserName string) string {
	if service, ok := chromeKeychainServices[browserName]; ok {
		return service
	}
	return browserName + " Safe Storage"
}

func loginKeychainPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		if u, err := user.Current(); err == nil {
			home = u.HomeDir
		}
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, "Library", "Keychains", "login.keychain-db")
}

func getKeychainPassword(browserName string) (string, error) {
	service := chromeKeychainService(browserName)
	loginKC := loginKeychainPath()

	accounts := []string{browserName, "Chrome", "Chromium", "Microsoft Edge", "Brave Browser", ""}

	type attempt struct {
		label string
		args  []string
	}

	var attempts []attempt
	for _, account := range accounts {
		base := []string{"find-generic-password", "-s", service}
		if account != "" {
			base = append(base, "-a", account)
		}
		base = append(base, "-w")

		attempts = append(attempts, attempt{
			label: fmt.Sprintf("service=%s account=%q default-keychain", service, account),
			args:  append([]string{}, base...),
		})
		if loginKC != "" {
			attempts = append(attempts, attempt{
				label: fmt.Sprintf("service=%s account=%q login-keychain", service, account),
				args:  append(append([]string{}, base...), loginKC),
			})
		}
		attempts = append(attempts, attempt{
			label: fmt.Sprintf("service=%s account=%q login-keychain-name", service, account),
			args: func() []string {
				args := []string{"find-generic-password", "-l", "login.keychain", "-s", service, "-w"}
				if account != "" {
					args = insertAccountFlag(args, account)
				}
				return args
			}(),
		})
	}

	var lastErr error
	for _, attempt := range attempts {
		cmd := exec.Command("security", attempt.args...)
		out, err := cmd.Output()
		if err != nil {
			lastErr = err
			continue
		}
		password := strings.TrimSpace(string(out))
		if password != "" {
			logf("keychain OK for %s via %s", browserName, attempt.label)
			return password, nil
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("empty keychain password")
	}
	return "", fmt.Errorf("keychain lookup failed for %s: %w", service, lastErr)
}

func insertAccountFlag(args []string, account string) []string {
	out := make([]string, 0, len(args)+2)
	out = append(out, args[0])
	for i := 1; i < len(args); i++ {
		if args[i] == "-s" {
			out = append(out, "-a", account)
		}
		out = append(out, args[i])
	}
	return out
}

func deriveChromeV10Key(password string) []byte {
	return pbkdf2.Key([]byte(password), []byte(darwinChromeSalt), darwinChromeIterations, darwinChromeKeyLen, sha1.New)
}

func decryptLocalStateEncryptedKey(keychainPassword, encKeyB64 string) []byte {
	if encKeyB64 == "" {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(encKeyB64)
	if err != nil || len(raw) < 4 {
		return nil
	}
	if string(raw[:3]) != "v10" {
		return nil
	}
	key := deriveChromeV10Key(keychainPassword)
	plain, err := aesCBCDecrypt(key, raw[3:])
	if err != nil || len(plain) == 0 {
		return nil
	}
	return plain
}

func ResolveKeys(cfg types.BrowserConfig) (*types.ResolvedKeys, error) {
	if cfg.IsFirefox {
		return &types.ResolvedKeys{}, nil
	}

	password, err := getKeychainPassword(cfg.Name)
	if err != nil {
		return nil, fmt.Errorf("could not get keychain password for %s: %w", cfg.Name, err)
	}

	keys := &types.ResolvedKeys{
		V10: deriveChromeV10Key(password),
	}

	localStatePath := browser.LocalStatePath(cfg)
	data, err := os.ReadFile(localStatePath)
	if err != nil {
		return keys, nil
	}

	var localState map[string]interface{}
	if err := json.Unmarshal(data, &localState); err != nil {
		return keys, nil
	}

	osCrypt, _ := localState["os_crypt"].(map[string]interface{})
	if osCrypt == nil {
		return keys, nil
	}

	if encKey, ok := osCrypt["encrypted_key"].(string); ok {
		if master := decryptLocalStateEncryptedKey(password, encKey); len(master) > 0 {
			keys.V10 = master
			logf("resolved %s master key from Local State encrypted_key (%d bytes)", cfg.Name, len(master))
		}
	}

	return keys, nil
}

func DecryptChromiumBlob(encrypted []byte, v10Key, v20Key []byte) string {
	if len(encrypted) == 0 {
		return ""
	}
	if len(encrypted) < 3 {
		return ""
	}

	prefix := string(encrypted[:3])
	switch prefix {
	case "v10", "v11":
		if plain := decryptDarwinV10CBC(encrypted, v10Key); plain != "" {
			return plain
		}
		return decryptDarwinGCM(encrypted, v10Key)
	case "v20":
		return decryptDarwinGCM(encrypted, v20Key)
	default:
		return ""
	}
}

func decryptDarwinV10CBC(encrypted, key []byte) string {
	if key == nil || len(key) == 0 {
		return ""
	}
	ciphertext := encrypted[3:]
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return ""
	}
	plaintext, err := aesCBCDecrypt(key, ciphertext)
	if err != nil {
		return ""
	}
	return CleanPassword(plaintext)
}

func decryptDarwinGCM(encrypted, key []byte) string {
	if key == nil || len(key) == 0 || len(encrypted) < 3+12+16 {
		return ""
	}
	nonce := encrypted[3:15]
	tag := encrypted[len(encrypted)-16:]
	ciphertext := encrypted[15 : len(encrypted)-16]
	plaintext, err := aesGCMDecrypt(key, nonce, ciphertext, tag)
	if err != nil {
		return ""
	}
	return CleanPassword(plaintext)
}

func aesCBCDecrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = 0x20
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	plaintext = pkcs5Unpad(plaintext)
	if plaintext == nil {
		return nil, fmt.Errorf("invalid padding")
	}
	return plaintext, nil
}

func aesGCMDecrypt(key, nonce, ciphertext, tag []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ctWithTag := make([]byte, len(ciphertext)+len(tag))
	copy(ctWithTag, ciphertext)
	copy(ctWithTag[len(ciphertext):], tag)
	return aesGCM.Open(nil, nonce, ctWithTag, nil)
}

func pkcs5Unpad(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > aes.BlockSize || padLen > len(data) {
		return nil
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if data[i] != byte(padLen) {
			return nil
		}
	}
	return data[:len(data)-padLen]
}

func CryptUnprotectData(in []byte) ([]byte, error) {
	return nil, fmt.Errorf("DPAPI not available on macOS")
}