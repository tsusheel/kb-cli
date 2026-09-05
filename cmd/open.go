package cmd

import (
	"fmt"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
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

		fmt.Printf("=========== [%s] ===========\n", n.ID[:7])
		if n.Note != "" {
			fmt.Printf("Note: %s\n", n.Note)
		}
		fmt.Printf("Type: %s | Status: %s", n.Type, n.Status)
		if n.Area != "" {
			fmt.Printf(" | Area: %s", n.Area)
		}
		fmt.Println()

		if !n.TargetDateTime.IsZero() {
			fmt.Printf("Due: %s\n", n.TargetDateTime.Format("2006-01-02 15:04"))
		}
		if n.Importance > 0 {
			fmt.Printf("Importance: %d/5 ", n.Importance)
		}
		if n.Clarity > 0 {
			fmt.Printf("Clarity: %d/5 ", n.Clarity)
		}
		if n.Source != "" {
			fmt.Printf("Source: %s", n.Source)
		}
		if n.Importance > 0 || n.Clarity > 0 || n.Source != "" {
			fmt.Println()
		}

		var tagStrs []string
		for _, t := range tags {
			tagStrs = append(tagStrs, t.Name)
		}
		if len(tagStrs) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(tagStrs, ", "))
		}

		if len(links) > 0 {
			fmt.Println("Links:")
			for _, l := range links {
				otherID := l.ToNote
				dir := "-->"
				if l.ToNote == n.ID {
					otherID = l.FromNote
					dir = "<--"
				}

				otherNoteDisplay := otherID[:7]
				if lNote, err := db.GetNote(otherID); err == nil && lNote.Note != "" {
					otherNoteDisplay = lNote.Note
				}
				fmt.Printf("  %s %s (%s)\n", dir, otherNoteDisplay, l.Type)
			}
		}

		fmt.Println("--------------------------------")
		fmt.Println(n.NoteFlesh)
		fmt.Println("========================================")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
}
