package utils

import (
	"os"
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
