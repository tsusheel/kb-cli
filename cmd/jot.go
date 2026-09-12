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
	jotType   string
	jotStatus string
	jotDue    string
	jotArea   string
)

func jotThought(text string, noteType string, noteStatus string, due string, area string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("cannot jot an empty thought")
	}

	var targetDT time.Time
	if due != "" {
		var err error
		targetDT, err = utils.ParseDate(due)
		if err != nil {
			return fmt.Errorf("invalid due date format %q: %w", due, err)
		}
	}

	if noteType == "" {
		noteType = string(models.DefaultNote)
	}
	if noteStatus == "" {
		noteStatus = string(models.Raw)
	}

	id := strings.ReplaceAll(uuid.New().String(), "-", "")
	n := &models.Note{
		ID:             id,
		Note:           text,
		NoteFlesh:      "",
		Type:           models.NoteType(noteType),
		Status:         models.Status(noteStatus),
		Area:           models.Area(area),
		TargetDateTime: targetDT,
	}

	if err := db.CreateNote(n); err != nil {
		return fmt.Errorf("failed to save jotted note: %w", err)
	}

	fmt.Printf("Jotted [%s] (%s): %s\n", n.ID[:7], n.Type, n.Note)
	return nil
}

var jotCmd = &cobra.Command{
	Use:     "jot [text]",
	Aliases: []string{"j", "q"},
	Short:   "Quickly jot down a fleeting thought without opening an editor",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := strings.Join(args, " ")
		return jotThought(text, jotType, jotStatus, jotDue, jotArea)
	},
}

func init() {
	jotCmd.Flags().StringVarP(&jotType, "type", "t", string(models.DefaultNote), "Note type (e.g., note, todo, idea, concept)")
	jotCmd.Flags().StringVarP(&jotStatus, "status", "s", string(models.Raw), "Status (default 'raw')")
	jotCmd.Flags().StringVarP(&jotDue, "due", "d", "", "Due / target date (e.g., 'today', 'tomorrow', 'monday', '+3d')")
	jotCmd.Flags().StringVarP(&jotArea, "area", "a", "", "Area (work, finance, personal)")
	rootCmd.AddCommand(jotCmd)
}
