package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
)

var (
	noteText   string
	noteType   string
	noteArea   string
	noteStatus string
	noteDue    string
	noteTags   []string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new note",
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetDT time.Time
		if noteDue != "" {
			var err error
			targetDT, err = ParseDate(noteDue)
			if err != nil {
				return fmt.Errorf("invalid due date %q: %w", noteDue, err)
			}
		}

		content, err := captureEditorContent("")
		if err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}

		content = strings.TrimSpace(content)
		if content == "" {
			fmt.Println("Note content is empty, aborting.")
			return nil
		}

		id := strings.ReplaceAll(uuid.New().String(), "-", "")

		n := &models.Note{
			ID:             id,
			Note:           noteText,
			NoteFlesh:      content,
			Type:           models.NoteType(noteType),
			Status:         models.Status(noteStatus),
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

		fmt.Printf("Successfully created note [%s]\n", id[:7])
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&noteText, "note", "n", "", "Summary or title of the note")
	addCmd.Flags().StringVar(&noteDue, "target", "", "Due / target date (e.g., 'today', 'tomorrow', 'monday', '+3d', or formatted date)")
	addCmd.Flags().StringVar(&noteType, "type", string(models.DefaultNote), "Type of the note")
	addCmd.Flags().StringVar(&noteArea, "area", "", "Area of the note")
	addCmd.Flags().StringVar(&noteStatus, "status", string(models.Active), "Status of the note")
	addCmd.Flags().StringSliceVar(&noteTags, "tags", []string{}, "Tags for the note")
	rootCmd.AddCommand(addCmd)
}
