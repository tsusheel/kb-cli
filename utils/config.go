package utils

import (
	"os"

	"github.com/spf13/viper"
)

// GetPostgresURL resolves the PostgreSQL connection URL from viper config or environment variables.
func GetPostgresURL() string {
	rawURL := viper.GetString("remote.postgres_url")
	if rawURL == "" {
		rawURL = viper.GetString("postgres_url")
	}
	if rawURL == "" {
		rawURL = os.Getenv("KB_POSTGRES_URL")
	}
	if rawURL == "" {
		rawURL = os.Getenv("DATABASE_URL")
	}
	return rawURL
}

// IsRemoteEnabled checks whether remote sync is enabled in configuration.
// Returns false if remote.enabled is explicitly false. Defaults to true if a postgres URL is present.
func IsRemoteEnabled() bool {
	if viper.IsSet("remote.enabled") {
		return viper.GetBool("remote.enabled")
	}
	return GetPostgresURL() != ""
}
