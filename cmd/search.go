package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
)

var searchCmd = &cobra.Command{
	Use:     "search [query]",
	Aliases: []string{"s", "find"},
	Short:   "Search notes using full-text search",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		notes, err := db.SearchNotes(query)
		if err != nil {
			return err
		}

		if len(notes) == 0 {
			fmt.Printf("No notes found matching: '%s'\n", query)
			return nil
		}

		printNotesTable(notes)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
