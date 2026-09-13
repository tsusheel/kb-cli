package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/utils"
)

var (
	noteText     string
	noteType     string
	noteArea     string
	noteStatus   string
	noteDue      string
	noteTags     []string
	noteEditFlag bool
)

var addCmd = &cobra.Command{
	Use:   "add [text]",
	Short: "Add a new note (inline text or in editor with -e/--edit)",
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetDT time.Time
		if noteDue != "" {
			var err error
			targetDT, err = utils.ParseDate(noteDue)
			if err != nil {
				return fmt.Errorf("invalid due date %q: %w", noteDue, err)
			}
		}

		title := strings.TrimSpace(strings.Join(args, " "))
		if title == "" && noteText != "" {
			title = strings.TrimSpace(noteText)
		}

		var flesh string
		if noteEditFlag || title == "" {
			var err error
			flesh, err = captureEditorContent("")
			if err != nil {
				return fmt.Errorf("failed to open editor: %w", err)
			}
			flesh = strings.TrimSpace(flesh)
			if flesh == "" && title == "" {
				fmt.Println("Note is empty, aborting.")
				return nil
			}
		}

		status := models.Status(noteStatus)
		if status == "" {
			if flesh != "" {
				status = models.Active
			} else {
				status = models.Raw
			}
		}

		id := strings.ReplaceAll(uuid.New().String(), "-", "")
		n := &models.Note{
			ID:             id,
			Note:           title,
			NoteFlesh:      flesh,
			Type:           models.NoteType(noteType),
			Status:         status,
			Area:           models.Area(noteArea),
			TargetDateTime: targetDT,
		}

		if err := db.CreateNote(n); err != nil {
			return fmt.Errorf("failed to save note: %w", err)
		}

		for _, tag := range noteTags {
			if err := db.AddTag(id, tag); err != nil {
				fmt.Printf("Warning: failed to add tag %s: %v\n", tag, err)
			}
		}

		displayTitle := n.Note
		if displayTitle == "" {
			displayTitle = "<Untitled>"
		}
		fmt.Printf("✔ Created note [%s] (%s): %s\n", utils.ShortID(id), n.Type, displayTitle)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&noteText, "note", "n", "", "Summary or title of the note")
	addCmd.Flags().StringVarP(&noteDue, "due", "d", "", "Due / target date (e.g., 'today', 'tomorrow', 'monday', '+3d')")
	addCmd.Flags().StringVarP(&noteType, "type", "t", string(models.DefaultNote), "Type of the note (e.g. note, todo, idea, project)")
	addCmd.Flags().StringVarP(&noteArea, "area", "a", "", "Area of the note (e.g. work, personal, finance)")
	addCmd.Flags().StringVarP(&noteStatus, "status", "s", "", "Status of the note (raw, active, refined, completed, archived)")
	addCmd.Flags().StringSliceVar(&noteTags, "tags", []string{}, "Tags for the note")
	addCmd.Flags().BoolVarP(&noteEditFlag, "edit", "e", false, "Open editor to write full note body (flesh)")

	addCmd.RegisterFlagCompletionFunc("type", completeNoteTypes)
	addCmd.RegisterFlagCompletionFunc("status", completeNoteStatuses)
	addCmd.RegisterFlagCompletionFunc("area", completeAreas)

	rootCmd.AddCommand(addCmd)
}

