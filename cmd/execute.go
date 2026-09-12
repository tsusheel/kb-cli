package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kb [thought]",
	Short: "Knowledge base CLI",
	Long:  "A fast CLI tool to manage your personal knowledge base and second brain.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func isKnownCommand(arg string) bool {
	if strings.HasPrefix(arg, "-") {
		return true // flags like --help, -h, --version
	}
	for _, c := range rootCmd.Commands() {
		if c.Name() == arg {
			return true
		}
		for _, alias := range c.Aliases {
			if alias == arg {
				return true
			}
		}
	}
	if arg == "help" || arg == "__complete" || arg == "__completeNoDesc" {
		return true
	}
	return false
}

func Execute() {
	args := os.Args[1:]
	if len(args) > 0 && !isKnownCommand(args[0]) {
		// First argument is not a known command or flag; route to 'jot'
		os.Args = append([]string{os.Args[0], "jot"}, args...)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
