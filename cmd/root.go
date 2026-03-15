package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new note",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Adding note")
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}

