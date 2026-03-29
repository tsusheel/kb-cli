package cmd

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
)

var (
	noteTitle  string
	noteType   string
	noteArea   string
	noteStatus string
	noteTags   []string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new note",
	RunE: func(cmd *cobra.Command, args []string) error {
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
			ID:      id,
			Title:   noteTitle,
			Content: content,
			Type:    models.NoteType(noteType),
			Status:  noteStatus,
			Area:    models.Area(noteArea),
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
	addCmd.Flags().StringVarP(&noteTitle, "title", "t", "", "Title of the note")
	addCmd.Flags().StringVar(&noteType, "type", string(models.DefaultNote), "Type of the note")
	addCmd.Flags().StringVar(&noteArea, "area", "", "Area of the note")
	addCmd.Flags().StringVar(&noteStatus, "status", string(models.Active), "Status of the note")
	addCmd.Flags().StringSliceVar(&noteTags, "tags", []string{}, "Tags for the note")
	rootCmd.AddCommand(addCmd)
}
