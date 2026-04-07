package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		notes, err := db.ListNotes()
		if err != nil {
			return err
		}

		if len(notes) == 0 {
			fmt.Println("No notes found.")
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
	rootCmd.AddCommand(listCmd)
}
