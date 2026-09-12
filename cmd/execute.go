package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/models"
)

var rootCmd = &cobra.Command{
	Use:   "kb [thought]",
	Short: "Knowledge base CLI",
	Long:  "A fast CLI tool to manage your personal knowledge base and second brain.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			thought := strings.Join(args, " ")
			return jotThought(thought, string(models.DefaultNote), string(models.Raw), "", "")
		}
		return cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

