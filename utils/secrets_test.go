package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

func TestNormalizeEnvKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"postgres.password", "KB_POSTGRES_PASSWORD"},
		{"postgres_password", "KB_POSTGRES_PASSWORD"},
		{"remote-db-pass", "KB_REMOTE_DB_PASS"},
		{"apiKey", "KB_APIKEY"},
	}

	for _, tc := range tests {
		got := NormalizeEnvKey(tc.input)
		if got != tc.expected {
			t.Errorf("NormalizeEnvKey(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestGetSecretEnvOverride(t *testing.T) {
	key := "test_secret_key"
	envVar := NormalizeEnvKey(key)

	os.Setenv(envVar, "test_super_secret_value")
	defer os.Unsetenv(envVar)

	val, err := GetSecret(key)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if val != "test_super_secret_value" {
		t.Errorf("GetSecret = %q, expected %q", val, "test_super_secret_value")
	}

	source := GetSecretSource(key)
	if !strings.Contains(source, "Environment") {
		t.Errorf("GetSecretSource = %q, expected containing Environment", source)
	}
}

func TestKeyringMock(t *testing.T) {
	keyring.MockInit()

	key := "mock_postgres_password"
	secret := "SuperSecretDatabasePassword123!"

	if err := SetSecret(key, secret); err != nil {
		t.Fatalf("SetSecret failed: %v", err)
	}

	if !HasSecret(key) {
		t.Errorf("expected HasSecret(%q) to be true", key)
	}

	val, err := GetSecret(key)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if val != secret {
		t.Errorf("GetSecret = %q, expected %q", val, secret)
	}

	if err := DeleteSecret(key); err != nil {
		t.Fatalf("DeleteSecret failed: %v", err)
	}

	if HasSecret(key) {
		t.Errorf("expected HasSecret(%q) to be false after deletion", key)
	}
}

func TestLocalEncryptedSecrets(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kb-secrets-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	viper.Set("base_path", tmpDir)
	defer viper.Set("base_path", "")

	key := "local_test_db_pass"
	secret := "P@ssw0rd!LocalEncrypted_999"

	// Test saving locally
	if err := setLocalEncryptedSecret(key, secret); err != nil {
		t.Fatalf("setLocalEncryptedSecret failed: %v", err)
	}

	// Verify file is encrypted on disk
	secretsFile := filepath.Join(tmpDir, ".secrets")
	rawContent, err := os.ReadFile(secretsFile)
	if err != nil {
		t.Fatalf("failed to read .secrets file: %v", err)
	}
	if strings.Contains(string(rawContent), secret) {
		t.Errorf("secret was found in plaintext in .secrets file!")
	}

	// Test reading locally
	retrieved, err := getLocalEncryptedSecret(key)
	if err != nil {
		t.Fatalf("getLocalEncryptedSecret failed: %v", err)
	}
	if retrieved != secret {
		t.Errorf("retrieved = %q, expected %q", retrieved, secret)
	}

	// Test deleting locally
	if err := deleteLocalEncryptedSecret(key); err != nil {
		t.Fatalf("deleteLocalEncryptedSecret failed: %v", err)
	}

	_, err = getLocalEncryptedSecret(key)
	if err == nil {
		t.Errorf("expected error after deleting local secret, got nil")
	}
}

func TestAESGCMEncryptionRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("Sensitive database password string 12345!@#$%^&*()")
	ciphertext, err := encryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("encryptAESGCM failed: %v", err)
	}

	decrypted, err := decryptAESGCM(key, ciphertext)
	if err != nil {
		t.Fatalf("decryptAESGCM failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted %q != plaintext %q", string(decrypted), string(plaintext))
	}

	// Test with invalid ciphertext
	_, err = decryptAESGCM(key, []byte("too short"))
	if err == nil {
		t.Errorf("expected error on short ciphertext, got nil")
	}
}
