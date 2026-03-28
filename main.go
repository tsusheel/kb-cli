package main 

import (
	"fmt"

	"github.com/tsusheel/kb-cli/cmd"
)

func main() {
	viper.SetConfigName(".kbconfig")
	viper.AddConfigPath("$HOME/.config/kb")
	viper.SetConfigType("yaml")
	
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("No config found")
	}

	cmd.Execute()
}

