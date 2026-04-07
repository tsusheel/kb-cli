package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

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

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, n := range notes {
			id := n.ID
			if len(id) > 7 {
				id = id[:7]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", id, n.Title, n.Type, n.Status, n.UpdatedAt.Format("2006-01-02 15:04"))
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
