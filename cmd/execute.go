package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cmd.Command{
	Use:	"kb",
	Short: 	"Knowledge base CLI",
	Long:	"A CLI tool to manage your personal knowledge base.",
	Run: func(cmd *cobra.Command, args []string){
		fmt.Println("Welcome to kb-cli")
	},
}

func Execute(){
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

