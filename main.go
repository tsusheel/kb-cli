package main 

import (
	"fmt"

	"github.com/tsusheel/kb-cli/cmd"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/app"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath("$HOME/.config/kb")
	viper.SetConfigType("yaml")

	app.InitApp()

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("No config found")
	}

	cmd.Execute()
}

