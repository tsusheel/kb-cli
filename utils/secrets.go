package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

const ServiceName = "kb-cli"

var secretsFileMutex sync.Mutex

// MaskURL hides sensitive passwords in database connection strings for safe display.
func MaskURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.User == nil {
		return rawURL
	}
	if _, hasPass := parsed.User.Password(); hasPass {
		parsed.User = url.UserPassword(parsed.User.Username(), "******")
		return parsed.String()
	}
	return rawURL
}

// NormalizeEnvKey converts keys like "postgres.password" or "db_password" to "KB_POSTGRES_PASSWORD"
func NormalizeEnvKey(key string) string {
	cleaned := strings.ReplaceAll(key, ".", "_")
	cleaned = strings.ReplaceAll(cleaned, "-", "_")
	return "KB_" + strings.ToUpper(cleaned)
}

func getSecretsDir() string {
	basePath := viper.GetString("base_path")
	if basePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".", ".kb")
		}
		basePath = filepath.Join(home, ".config", "kb")
	}
	return basePath
}

func getOrCreateMasterKey() ([]byte, error) {
	dir := getSecretsDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	keyPath := filepath.Join(dir, ".secrets.key")

	if keyData, err := os.ReadFile(keyPath); err == nil && len(keyData) == 32 {
		return keyData, nil
	}

	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	if err := os.WriteFile(keyPath, newKey, 0600); err != nil {
		return nil, fmt.Errorf("failed to save encryption key: %w", err)
	}
	return newKey, nil
}

func encryptAESGCM(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func decryptAESGCM(key []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	return plaintext, nil
}

func loadLocalEncryptedSecrets() (map[string]string, error) {
	secretsFileMutex.Lock()
	defer secretsFileMutex.Unlock()

	secretsPath := filepath.Join(getSecretsDir(), ".secrets")
	if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
		return make(map[string]string), nil
	}

	cipherData, err := os.ReadFile(secretsPath)
	if err != nil {
		return nil, err
	}
	if len(cipherData) == 0 {
		return make(map[string]string), nil
	}

	key, err := getOrCreateMasterKey()
	if err != nil {
		return nil, err
	}

	plainData, err := decryptAESGCM(key, cipherData)
	if err != nil {
		return nil, err
	}

	var secrets map[string]string
	if err := json.Unmarshal(plainData, &secrets); err != nil {
		return nil, err
	}
	if secrets == nil {
		secrets = make(map[string]string)
	}
	return secrets, nil
}

func saveLocalEncryptedSecrets(secrets map[string]string) error {
	secretsFileMutex.Lock()
	defer secretsFileMutex.Unlock()

	dir := getSecretsDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	key, err := getOrCreateMasterKey()
	if err != nil {
		return err
	}

	plainData, err := json.Marshal(secrets)
	if err != nil {
		return err
	}

	cipherData, err := encryptAESGCM(key, plainData)
	if err != nil {
		return err
	}

	secretsPath := filepath.Join(dir, ".secrets")
	return os.WriteFile(secretsPath, cipherData, 0600)
}

func setLocalEncryptedSecret(key, secret string) error {
	secrets, err := loadLocalEncryptedSecrets()
	if err != nil {
		secrets = make(map[string]string)
	}
	secrets[key] = secret
	return saveLocalEncryptedSecrets(secrets)
}

func getLocalEncryptedSecret(key string) (string, error) {
	secrets, err := loadLocalEncryptedSecrets()
	if err != nil {
		return "", err
	}
	val, ok := secrets[key]
	if !ok || val == "" {
		return "", fmt.Errorf("secret %q not found in local encrypted vault", key)
	}
	return val, nil
}

func deleteLocalEncryptedSecret(key string) error {
	secrets, err := loadLocalEncryptedSecrets()
	if err != nil {
		return nil
	}
	if _, ok := secrets[key]; !ok {
		return nil
	}
	delete(secrets, key)
	return saveLocalEncryptedSecrets(secrets)
}

// SetSecret saves a secret in the OS Keyring, falling back to local AES-256-GCM encrypted vault if keyring fails.
func SetSecret(key string, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return fmt.Errorf("cannot set an empty secret")
	}

	// 1. Try OS Keyring
	err := keyring.Set(ServiceName, key, secret)
	if err == nil {
		// Clean up any fallback entry if present
		_ = deleteLocalEncryptedSecret(key)
		return nil
	}

	// 2. Fallback to Local Encrypted Store
	if fallbackErr := setLocalEncryptedSecret(key, secret); fallbackErr != nil {
		return fmt.Errorf("failed storing secret in OS Keyring (%v) and fallback vault (%v)", err, fallbackErr)
	}
	return nil
}

// GetSecret retrieves a secret, checking Environment Variables first, then OS Keyring, then local encrypted vault.
func GetSecret(key string) (string, error) {
	// 1. Check environment variable override
	envVar := NormalizeEnvKey(key)
	if val := os.Getenv(envVar); val != "" {
		return val, nil
	}

	// 2. Query OS Keyring
	val, err := keyring.Get(ServiceName, key)
	if err == nil && val != "" {
		return val, nil
	}

	// 3. Fallback: Query local encrypted vault
	if localVal, localErr := getLocalEncryptedSecret(key); localErr == nil && localVal != "" {
		return localVal, nil
	}

	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("secret %q not found", key)
}

// GetSecretSource returns a descriptive source of where the secret is configured.
func GetSecretSource(key string) string {
	envVar := NormalizeEnvKey(key)
	if os.Getenv(envVar) != "" {
		return "Environment (" + envVar + ")"
	}
	if val, err := keyring.Get(ServiceName, key); err == nil && val != "" {
		return "OS Keyring"
	}
	if val, err := getLocalEncryptedSecret(key); err == nil && val != "" {
		return "Local Secure Vault"
	}
	return ""
}

// DeleteSecret removes a secret from OS Keyring and local encrypted vault.
func DeleteSecret(key string) error {
	keyringErr := keyring.Delete(ServiceName, key)
	localErr := deleteLocalEncryptedSecret(key)

	if keyringErr == nil || localErr == nil {
		return nil
	}
	return keyringErr
}

// HasSecret returns true if a secret is available via environment variable, OS Keyring, or local encrypted vault.
func HasSecret(key string) bool {
	_, err := GetSecret(key)
	return err == nil
}

// PromptSecret prompts the user for sensitive input with terminal masking (hidden keystrokes).
func PromptSecret(promptText string) (string, error) {
	fmt.Print(promptText)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Print newline after masked input
	if err != nil {
		return "", fmt.Errorf("failed to read secret input: %w", err)
	}
	return strings.TrimSpace(string(bytePassword)), nil
}
