package utils

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

const ServiceName = "kb-cli"

// NormalizeEnvKey converts keys like "postgres.password" or "db_password" to "KB_POSTGRES_PASSWORD"
func NormalizeEnvKey(key string) string {
	cleaned := strings.ReplaceAll(key, ".", "_")
	cleaned = strings.ReplaceAll(cleaned, "-", "_")
	return "KB_" + strings.ToUpper(cleaned)
}

// SetSecret saves a secret in the OS Keyring.
func SetSecret(key string, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return fmt.Errorf("cannot set an empty secret")
	}
	return keyring.Set(ServiceName, key, secret)
}

// GetSecret retrieves a secret, checking Environment Variables first, then OS Keyring.
func GetSecret(key string) (string, error) {
	// 1. Check environment variable override
	envVar := NormalizeEnvKey(key)
	if val := os.Getenv(envVar); val != "" {
		return val, nil
	}

	// 2. Query OS Keyring
	val, err := keyring.Get(ServiceName, key)
	if err != nil {
		return "", err
	}
	return val, nil
}

// DeleteSecret removes a secret from the OS Keyring.
func DeleteSecret(key string) error {
	return keyring.Delete(ServiceName, key)
}

// HasSecret returns true if a secret is available via environment variable or OS Keyring.
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
