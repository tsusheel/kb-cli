package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/utils"
)

var (
	editTitle      string
	editType       string
	editStatus     string
	editArea       string
	editDue        string
	openEditorFlag bool
)

var editCmd = &cobra.Command{
	Use:     "edit <id> [new_title]",
	Aliases: []string{"flesh", "rename", "update"},
	Short:   "Edit a note's title, body (flesh), or metadata",
	Long: `Edit an existing note.

Examples:
  # Update title directly:
  kb edit f59a66e "today's note"
  kb rename f59a66e "today's note"

  # Update title and metadata flags:
  kb edit f59a66e -n "today's note" --status completed --type project

  # Open $EDITOR to edit detailed body (note flesh):
  kb edit f59a66e
  kb flesh f59a66e`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		n, err := db.GetNote(id)
		if err != nil {
			return err
		}

		hasPositionalTitle := len(args) == 2
		if hasPositionalTitle {
			editTitle = args[1]
		}

		hasFlagUpdates := editTitle != "" || editType != "" || editStatus != "" || editArea != "" || editDue != ""

		if editTitle != "" {
			n.Note = strings.TrimSpace(editTitle)
		}
		if editType != "" {
			n.Type = models.NoteType(editType)
		}
		if editStatus != "" {
			n.Status = models.Status(editStatus)
		}
		if editArea != "" {
			n.Area = models.Area(editArea)
		}
		if editDue != "" {
			if editDue == "clear" || editDue == "none" {
				n.TargetDateTime = time.Time{}
			} else {
				targetDT, err := utils.ParseDate(editDue)
				if err != nil {
					return fmt.Errorf("invalid due date %q: %w", editDue, err)
				}
				n.TargetDateTime = targetDT
			}
		}

		// Open editor if explicitly requested OR if no title/flags were provided
		if openEditorFlag || (!hasPositionalTitle && !hasFlagUpdates) {
			updatedContent, err := captureEditorContent(n.NoteFlesh)
			if err != nil {
				return fmt.Errorf("failed to open editor: %w", err)
			}
			n.NoteFlesh = strings.TrimSpace(updatedContent)
		}

		if err := db.UpdateNote(n); err != nil {
			return fmt.Errorf("failed to update note: %w", err)
		}

		fmt.Printf("Successfully updated note [%s] %s\n", n.ID[:7], n.Note)
		return nil
	},
}

func init() {
	editCmd.Flags().StringVarP(&editTitle, "note", "n", "", "New title or summary of the note")
	editCmd.Flags().StringVarP(&editTitle, "title", "", "", "Alias for --note")
	editCmd.Flags().StringVarP(&editType, "type", "t", "", "New note type (e.g. todo, project, note, idea)")
	editCmd.Flags().StringVarP(&editStatus, "status", "s", "", "New status (active, raw, refined, completed, archived)")
	editCmd.Flags().StringVarP(&editArea, "area", "a", "", "New area (work, personal, finance)")
	editCmd.Flags().StringVarP(&editDue, "due", "d", "", "New due date or target date ('clear' to remove)")
	editCmd.Flags().BoolVarP(&openEditorFlag, "editor", "e", false, "Open editor to edit body even when updating title")

	editCmd.ValidArgsFunction = completeNoteIDs
	editCmd.RegisterFlagCompletionFunc("type", completeNoteTypes)
	editCmd.RegisterFlagCompletionFunc("status", completeNoteStatuses)
	editCmd.RegisterFlagCompletionFunc("area", completeAreas)

	rootCmd.AddCommand(editCmd)
}
