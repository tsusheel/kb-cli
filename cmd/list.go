package cmd

import (
	"fmt"
	"os"
	"strings"

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
	Use:     "list [query]",
	Aliases: []string{"ls", "search", "find"},
	Short:   "List notes or search by keyword (e.g. 'kb ls', 'kb ls postgres', 'kb ls -t todo')",
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

		query := strings.TrimSpace(strings.Join(args, " "))

		var notes []models.Note
		var err error

		if query != "" {
			notes, err = db.SearchNotesExtended(query, filterType, listStatusFlag, listAreaFlag)
			if err != nil {
				return err
			}
			if len(notes) == 0 {
				fmt.Printf("No notes found matching: '%s'\n", query)
				return nil
			}
		} else {
			notes, err = db.ListNotesExtended(filterType, listStatusFlag, listAreaFlag, false)
			if err != nil {
				return err
			}
			if len(notes) == 0 {
				fmt.Println("No notes found.")
				return nil
			}
		}

		printNotesTable(notes)
		return nil
	},
}

func printNotesTable(notes []models.Note) {
	utils.RenderNotesTable(notes, os.Stdout)
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
