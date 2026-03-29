package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/tsusheel/kb-cli/app"
	"github.com/tsusheel/kb-cli/cmd"
)

func main() {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "kb")
	configFile := filepath.Join(configDir, "config.yaml")

	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")

	// Try reading config
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("Config not found, creating default config...")

		// 1. Create directory if not exists
		if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
			panic(err)
		}

		// 2. Set default values
		viper.Set("app_name", "kb-app")

		// 3. Create the config file
		file, err := os.Create(configFile)
		if err != nil {
			panic(err)
		}
		file.Close()

		// 4. Write defaults to file
		if err := viper.WriteConfigAs(configFile); err != nil {
			panic(err)
		}

		fmt.Println("Default config created at:", configFile)
	}

	app.InitApp()

	cmd.Execute()
}
