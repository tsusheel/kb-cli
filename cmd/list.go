package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/utils"
)

var (
	listNotesFlag    bool
	listProjectsFlag bool
	listTodosFlag    bool
	listTypeFlag     string
	listStatusFlag   string
	listAreaFlag     string
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all items",
	RunE: func(cmd *cobra.Command, args []string) error {
		var filterType string
		if listTypeFlag != "" {
			filterType = listTypeFlag
		} else if listNotesFlag {
			filterType = "note"
		} else if listProjectsFlag {
			filterType = "project"
		} else if listTodosFlag {
			filterType = "todo"
		}

		notes, err := db.ListNotesExtended(filterType, listStatusFlag, listAreaFlag, false)
		if err != nil {
			return err
		}

		if len(notes) == 0 {
			fmt.Println("No notes found.")
			return nil
		}

		printNotesTable(notes)
		return nil
	},
}

func printNotesTable(notes []models.Note) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, n := range notes {
		displayNote := n.Note
		if displayNote == "" {
			displayNote = "<Untitled>"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", utils.ShortID(n.ID), displayNote, n.Type, n.Status, n.UpdatedAt.Format("2006-01-02 15:04"))
	}
	w.Flush()
}

func init() {
	listCmd.Flags().BoolVarP(&listNotesFlag, "notes", "n", false, "List only notes")
	listCmd.Flags().BoolVarP(&listProjectsFlag, "projects", "p", false, "List only projects")
	listCmd.Flags().BoolVarP(&listTodosFlag, "todos", "d", false, "List only todos")
	listCmd.Flags().StringVarP(&listTypeFlag, "type", "t", "", "Filter by note type (e.g. todo, project, note, idea)")
	listCmd.Flags().StringVarP(&listStatusFlag, "status", "s", "", "Filter by status (e.g. active, raw, refined, completed, archived)")
	listCmd.Flags().StringVarP(&listAreaFlag, "area", "a", "", "Filter by area (e.g. work, personal, finance)")

	listCmd.RegisterFlagCompletionFunc("type", completeNoteTypes)
	listCmd.RegisterFlagCompletionFunc("status", completeNoteStatuses)
	listCmd.RegisterFlagCompletionFunc("area", completeAreas)
	rootCmd.AddCommand(listCmd)
}
