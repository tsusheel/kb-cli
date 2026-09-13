package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
)

func restoreSingleItem(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id cannot be empty")
	}

	// 1. Try restoring as Note
	if n, err := db.GetNote(id); err == nil {
		if n.DeletedAt.IsZero() {
			return fmt.Errorf("note [%s] is not deleted", n.ID[:7])
		}
		if _, err := db.RestoreNote(n.ID); err != nil {
			return fmt.Errorf("failed restoring note [%s]: %w", n.ID[:7], err)
		}
		fmt.Printf("✔ Restored note [%s] %s\n", n.ID[:7], n.Note)
		return nil
	}

	// 2. Try restoring as Daily Log
	if l, err := db.GetDailyLog(id); err == nil {
		if l.DeletedAt.IsZero() {
			return fmt.Errorf("daily log [%s] is not deleted", l.ID[:7])
		}
		if _, err := db.RestoreDailyLog(l.ID); err != nil {
			return fmt.Errorf("failed restoring daily log [%s]: %w", l.ID[:7], err)
		}
		logContent := l.Content
		if len(logContent) > 40 {
			logContent = logContent[:37] + "..."
		}
		fmt.Printf("✔ Restored daily log [%s] %s\n", l.ID[:7], logContent)
		return nil
	}

	return fmt.Errorf("no note or daily log found matching ID %q", id)
}

var restoreCmd = &cobra.Command{
	Use:     "restore <id...>",
	Aliases: []string{"undelete", "untrash"},
	Short:   "Restore a soft-deleted note or daily log",
	Long: `Restore a soft-deleted note or daily log back to active state by clearing deleted_at.
Also records a 'restored' entry in the audit history.

Examples:
  # Restore a note by ID:
  kb restore 3996f5f
  kb undelete 3996f5f`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, id := range args {
			if err := restoreSingleItem(id); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	restoreCmd.ValidArgsFunction = completeNoteIDs
	rootCmd.AddCommand(restoreCmd)
}
