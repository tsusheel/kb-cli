package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
)

var editCmd = &cobra.Command{
	Use:     "edit <id>",
	Aliases: []string{"flesh"},
	Short:   "Open editor to write or edit note flesh (detailed body)",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		n, err := db.GetNote(id)
		if err != nil {
			return err
		}

		updatedContent, err := captureEditorContent(n.NoteFlesh)
		if err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}

		updatedContent = strings.TrimSpace(updatedContent)
		n.NoteFlesh = updatedContent

		if err := db.UpdateNote(n); err != nil {
			return fmt.Errorf("failed to update note: %w", err)
		}

		fmt.Printf("Successfully updated note flesh for [%s] %s\n", n.ID[:7], n.Note)
		return nil
	},
}

func init() {
	editCmd.ValidArgsFunction = completeNoteIDs
	rootCmd.AddCommand(editCmd)
}
