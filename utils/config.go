package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
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

// DeleteConfigKey removes a configuration key from config.yaml and reloads Viper in-memory state.
func DeleteConfigKey(key string) (bool, error) {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return false, err
		}
		configFile = filepath.Join(home, ".config", "kb", "config.yaml")
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return false, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return false, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	var rawMap map[string]interface{}
	if err := yaml.Unmarshal(data, &rawMap); err != nil {
		return false, fmt.Errorf("failed to parse yaml config: %w", err)
	}
	if rawMap == nil {
		return false, nil
	}

	parts := strings.Split(key, ".")
	deleted := deleteNestedMapKey(rawMap, parts)
	if !deleted {
		return false, nil
	}

	outData, err := yaml.Marshal(rawMap)
	if err != nil {
		return false, fmt.Errorf("failed to serialize yaml config: %w", err)
	}

	if err := os.WriteFile(configFile, outData, 0644); err != nil {
		return false, fmt.Errorf("failed to write updated config to %s: %w", configFile, err)
	}

	// Reload viper configuration
	viper.Reset()
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("KB")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	return true, nil
}

func deleteNestedMapKey(val interface{}, parts []string) bool {
	if len(parts) == 0 {
		return false
	}
	key := parts[0]

	switch m := val.(type) {
	case map[string]interface{}:
		if len(parts) == 1 {
			for k := range m {
				if strings.EqualFold(k, key) {
					delete(m, k)
					return true
				}
			}
			return false
		}
		for k, v := range m {
			if strings.EqualFold(k, key) {
				deleted := deleteNestedMapKey(v, parts[1:])
				if subMap, ok := v.(map[string]interface{}); ok && len(subMap) == 0 {
					delete(m, k)
				}
				return deleted
			}
		}
	case map[interface{}]interface{}:
		if len(parts) == 1 {
			for k := range m {
				if strK, ok := k.(string); ok && strings.EqualFold(strK, key) {
					delete(m, k)
					return true
				}
			}
			return false
		}
		for k, v := range m {
			if strK, ok := k.(string); ok && strings.EqualFold(strK, key) {
				deleted := deleteNestedMapKey(v, parts[1:])
				if subMap, ok := v.(map[interface{}]interface{}); ok && len(subMap) == 0 {
					delete(m, k)
				}
				return deleted
			}
		}
	}
	return false
}
