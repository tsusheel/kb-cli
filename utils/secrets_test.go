package utils

import (
	"os"
	"testing"

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
}

func TestKeyringMock(t *testing.T) {
	keyring.MockInit()

	key := "postgres_password"
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
