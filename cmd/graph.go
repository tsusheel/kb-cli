package cmd

import (
	"fmt"
	"os"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/utils"
)

var backlinksCmd = &cobra.Command{
	Use:     "backlinks <id>",
	Aliases: []string{"incoming", "bl"},
	Short:   "View incoming backlinks pointing to a note",
	Long: `Display all incoming links that reference a specific note.

Examples:
  # View backlinks for a note:
  kb backlinks a1b2c3d
  kb bl a1b2c3d`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		n, err := db.GetNote(id)
		if err != nil {
			return fmt.Errorf("note %q not found: %w", id, err)
		}

		links, err := db.GetIncomingLinks(n.ID)
		if err != nil {
			return fmt.Errorf("failed fetching backlinks: %w", err)
		}

		noteMap := make(map[string]*models.Note)
		for _, l := range links {
			if _, ok := noteMap[l.FromNote]; !ok {
				if fromNote, err := db.GetNote(l.FromNote); err == nil {
					noteMap[l.FromNote] = fromNote
				}
			}
		}

		fmt.Printf("=== Backlinks pointing to [%s] %q ===\n", utils.ShortID(n.ID), n.Note)
		utils.RenderBacklinksTable(links, noteMap, os.Stdout)
		return nil
	},
}

var graphCmd = &cobra.Command{
	Use:     "graph [id]",
	Aliases: []string{"relations", "links"},
	Short:   "Visualize relational graph connections for a note or entire knowledge base",
	Long: `Visualize incoming and outgoing relationships for a note as an ASCII tree.

If ID is omitted, launches interactive fuzzy-finder to select a note.

Examples:
  # View connections for a note:
  kb graph a1b2c3d

  # Interactive fuzzy-finder graph:
  kb graph`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetID string
		if len(args) == 1 {
			targetID = args[0]
		} else {
			items, err := GetActiveCLIItems()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Println("No active notes found.")
				return nil
			}

			idx, err := fuzzyfinder.Find(items, func(i int) string {
				return items[i].FormatFuzzy()
			}, fuzzyfinder.WithPreviewWindow(func(i int, width, height int) string {
				if i < 0 || i >= len(items) {
					return ""
				}
				return items[i].RenderPreview(width)
			}))
			if err != nil {
				if err == fuzzyfinder.ErrAbort {
					return nil
				}
				return err
			}
			targetID = items[idx].ID
		}

		n, err := db.GetNote(targetID)
		if err != nil {
			return fmt.Errorf("note %q not found: %w", targetID, err)
		}

		outgoing, err := db.GetOutgoingLinks(n.ID)
		if err != nil {
			return err
		}
		incoming, err := db.GetIncomingLinks(n.ID)
		if err != nil {
			return err
		}

		noteMap := make(map[string]*models.Note)
		for _, l := range outgoing {
			if _, ok := noteMap[l.ToNote]; !ok {
				if toNote, err := db.GetNote(l.ToNote); err == nil {
					noteMap[l.ToNote] = toNote
				}
			}
		}
		for _, l := range incoming {
			if _, ok := noteMap[l.FromNote]; !ok {
				if fromNote, err := db.GetNote(l.FromNote); err == nil {
					noteMap[l.FromNote] = fromNote
				}
			}
		}

		fmt.Println("=== Relational Graph View ===")
		utils.RenderGraphTree(n, outgoing, incoming, noteMap, os.Stdout)
		return nil
	},
}

var orphansCmd = &cobra.Command{
	Use:     "orphans",
	Aliases: []string{"isolated"},
	Short:   "List unlinked and untagged notes requiring connection or categorization",
	RunE: func(cmd *cobra.Command, args []string) error {
		orphans, err := db.GetOrphanNotes()
		if err != nil {
			return fmt.Errorf("failed fetching orphan notes: %w", err)
		}

		fmt.Printf("=== Orphan Notes (%d) ===\n", len(orphans))
		utils.RenderOrphansTable(orphans, os.Stdout)
		if len(orphans) > 0 {
			fmt.Println("\nTip: Use 'kb edit <id>' to link or tag these notes into your knowledge graph.")
		}
		return nil
	},
}

func init() {
	backlinksCmd.ValidArgsFunction = completeNoteIDs
	graphCmd.ValidArgsFunction = completeNoteIDs

	rootCmd.AddCommand(backlinksCmd)
	rootCmd.AddCommand(graphCmd)
	rootCmd.AddCommand(orphansCmd)
}
