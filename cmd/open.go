package cmd

import (
	"fmt"
	"os"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/utils"
)

var openCmd = &cobra.Command{
	Use:     "open [id]",
	Aliases: []string{"view"},
	Short:   "View a note. Uses fuzzy-finder if id is omitted.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var id string

		if len(args) > 0 {
			id = args[0]
		} else {
			notes, err := db.ListNotes("")
			if err != nil {
				return err
			}
			if len(notes) == 0 {
				fmt.Println("No notes found.")
				return nil
			}

			idx, err := fuzzyfinder.Find(notes, func(i int) string {
				displayTitle := notes[i].Note
				if displayTitle == "" {
					displayTitle = "<Untitled>"
				}
				return fmt.Sprintf("[%s] %s (%s)", notes[i].ID[:7], displayTitle, notes[i].Type)
			})
			if err != nil {
				if err == fuzzyfinder.ErrAbort {
					return nil
				}
				return err
			}
			id = notes[idx].ID
		}

		n, err := db.GetNote(id)
		if err != nil {
			return err
		}

		tags, err := db.GetTagsForNote(n.ID)
		if err != nil {
			return err
		}

		links, err := db.GetLinksForNote(n.ID)
		if err != nil {
			return err
		}

		utils.RenderNoteDetail(n, tags, links, os.Stdout)
		return nil
	},
}

func init() {
	openCmd.ValidArgsFunction = completeNoteIDs
	rootCmd.AddCommand(openCmd)
}
