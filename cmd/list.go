package cmd

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
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

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"ID", "Title", "Type", "Status", "Updated At"})

		for _, n := range notes {
			id := n.ID
			if len(id) > 7 {
				id = id[:7]
			}
			t.AppendRow(table.Row{id, n.Title, n.Type, n.Status, n.UpdatedAt.Format("2004-07-31 17:30")})
		}

		t.Render()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
