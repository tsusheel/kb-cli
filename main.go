package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/tsusheel/kb-cli/app"
	"github.com/tsusheel/kb-cli/cmd"
)

func resolveConfigPaths() (string, string) {
	if envConfig := os.Getenv("KB_CONFIG"); envConfig != "" {
		return filepath.Dir(envConfig), envConfig
	}
	for i, arg := range os.Args {
		if (arg == "--config" || arg == "-c") && i+1 < len(os.Args) {
			customFile := os.Args[i+1]
			return filepath.Dir(customFile), customFile
		}
		if strings.HasPrefix(arg, "--config=") {
			customFile := strings.TrimPrefix(arg, "--config=")
			return filepath.Dir(customFile), customFile
		}
	}
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "kb")
	return configDir, filepath.Join(configDir, "config.yaml")
}

func main() {
	configDir, configFile := resolveConfigPaths()

	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("KB")
	viper.AutomaticEnv()

	// Try reading config
	err := viper.ReadInConfig()
	if err != nil {
		// 1. Create directory if not exists
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config directory %s: %v\n", configDir, err)
			os.Exit(1)
		}

		// 2. Set default values
		viper.Set("app_name", "kb-app")
		viper.Set("base_path", configDir)
		viper.Set("date_format", "2006-01-02")

		// 3. Create the config file
		file, err := os.Create(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config file %s: %v\n", configFile, err)
			os.Exit(1)
		}
		file.Close()

		// 4. Write defaults to file
		if err := viper.WriteConfigAs(configFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing config defaults to %s: %v\n", configFile, err)
			os.Exit(1)
		}
	}

	app.InitApp()

	cmd.Execute()
}
