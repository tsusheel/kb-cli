package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestGetPostgresURL(t *testing.T) {
	// 1. Remote postgres URL in viper
	viper.Set("remote.postgres_url", "postgresql://user:pass@localhost:5432/db1")
	if GetPostgresURL() != "postgresql://user:pass@localhost:5432/db1" {
		t.Errorf("expected URL from remote.postgres_url, got %s", GetPostgresURL())
	}

	// 2. Fallback to postgres_url
	viper.Set("remote.postgres_url", "")
	viper.Set("postgres_url", "postgresql://user:pass@localhost:5432/db2")
	if GetPostgresURL() != "postgresql://user:pass@localhost:5432/db2" {
		t.Errorf("expected URL from postgres_url, got %s", GetPostgresURL())
	}

	// 3. Fallback to KB_POSTGRES_URL environment variable
	viper.Set("postgres_url", "")
	os.Setenv("KB_POSTGRES_URL", "postgresql://user:pass@localhost:5432/db3")
	defer os.Unsetenv("KB_POSTGRES_URL")
	if GetPostgresURL() != "postgresql://user:pass@localhost:5432/db3" {
		t.Errorf("expected URL from KB_POSTGRES_URL, got %s", GetPostgresURL())
	}
}

func TestIsRemoteEnabled(t *testing.T) {
	// 1. Explicitly false
	viper.Set("remote.enabled", false)
	if IsRemoteEnabled() {
		t.Errorf("expected IsRemoteEnabled() to be false when remote.enabled is false")
	}

	// 2. Explicitly true
	viper.Set("remote.enabled", true)
	if !IsRemoteEnabled() {
		t.Errorf("expected IsRemoteEnabled() to be true when remote.enabled is true")
	}
}

func TestDeleteConfigKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kb-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "config.yaml")
	initialContent := `app_name: kb-test
test_key: sample_value
remote:
  enabled: true
  postgres_url: postgresql://localhost:5432/testdb
`
	if err := os.WriteFile(configFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	viper.Reset()
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("failed to read in config: %v", err)
	}

	// 1. Delete top-level key
	deleted, err := DeleteConfigKey("test_key")
	if err != nil {
		t.Fatalf("unexpected error deleting test_key: %v", err)
	}
	if !deleted {
		t.Errorf("expected test_key to be reported as deleted")
	}
	if viper.IsSet("test_key") {
		t.Errorf("expected test_key to be unset in viper")
	}

	// 2. Delete nested key
	deleted, err = DeleteConfigKey("remote.enabled")
	if err != nil {
		t.Fatalf("unexpected error deleting remote.enabled: %v", err)
	}
	if !deleted {
		t.Errorf("expected remote.enabled to be reported as deleted")
	}
	if viper.IsSet("remote.enabled") {
		t.Errorf("expected remote.enabled to be unset in viper")
	}
	if !viper.IsSet("remote.postgres_url") {
		t.Errorf("expected remote.postgres_url to still be present")
	}

	// 3. Delete non-existent key
	deleted, err = DeleteConfigKey("non_existent_key")
	if err != nil {
		t.Fatalf("unexpected error deleting non_existent_key: %v", err)
	}
	if deleted {
		t.Errorf("expected non_existent_key to return false for deleted")
	}
}
