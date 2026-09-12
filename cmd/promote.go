package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/utils"
)

var (
	promoteType   string
	promoteStatus string
)

var promoteCmd = &cobra.Command{
	Use:   "promote <log_id>",
	Short: "Promote a daily log entry into a permanent note (non-interactive)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logID := args[0]

		n, err := db.PromoteDailyLog(logID, models.NoteType(promoteType), models.Status(promoteStatus))
		if err != nil {
			return err
		}

		fmt.Printf("Successfully promoted log [%s] -> Note [%s] (%s): %s\n", utils.ShortID(logID), utils.ShortID(n.ID), n.Type, n.Note)
		return nil
	},
}

func init() {
	promoteCmd.ValidArgsFunction = completeUnpromotedLogIDs
	promoteCmd.Flags().StringVarP(&promoteType, "type", "t", string(models.DefaultNote), "Note type for the promoted note (e.g., note, todo, idea, project)")
	promoteCmd.Flags().StringVarP(&promoteStatus, "status", "s", string(models.Raw), "Status for the promoted note (default 'raw')")
	promoteCmd.RegisterFlagCompletionFunc("type", completeNoteTypes)
	promoteCmd.RegisterFlagCompletionFunc("status", completeNoteStatuses)
	rootCmd.AddCommand(promoteCmd)
}
