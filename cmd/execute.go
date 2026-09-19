package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "kb [thought]",
	Short:         "Knowledge base CLI",
	Long:          "A fast CLI tool to manage your personal knowledge base and second brain.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var configFileFlag string

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFileFlag, "config", "c", "", "Path to custom config.yaml profile")
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
	if arg == "help" || arg == "completion" || arg == "__complete" || arg == "__completeNoDesc" {
		return true
	}
	return false
}

func findFirstPositionalArg(args []string) (int, string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if (arg == "--config" || arg == "-c") && i+1 < len(args) {
			i++ // skip flag argument
			continue
		}
		if strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "-c=") {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return i, arg
	}
	return -1, ""
}

func Execute() {
	args := os.Args[1:]
	if posIdx, firstPos := findFirstPositionalArg(args); posIdx >= 0 && !isKnownCommand(firstPos) {
		// Insert "add" subcommand right before the first positional argument
		actualPos := posIdx + 1
		newArgs := make([]string, 0, len(os.Args)+1)
		newArgs = append(newArgs, os.Args[:actualPos]...)
		newArgs = append(newArgs, "add")
		newArgs = append(newArgs, os.Args[actualPos:]...)
		os.Args = newArgs
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
