package cmd

import (
	"fmt"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
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
	Use:     "restore [id...]",
	Aliases: []string{"undelete", "untrash"},
	Short:   "Restore a soft-deleted note or daily log (fuzzy-select if ID is omitted)",
	Long: `Restore a soft-deleted note or daily log back to active state by clearing deleted_at.
When run without arguments, launches an interactive multi-selection fuzzy finder containing all soft-deleted items.
Also records a 'restored' entry in the audit history.

Examples:
  # Restore a note by ID:
  kb restore 3996f5f
  kb undelete 3996f5f

  # Fuzzy-find and select deleted notes/logs to restore:
  kb restore`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			items, err := GetDeletedCLIItems()
			if err != nil {
				return err
			}

			if len(items) == 0 {
				fmt.Println("No soft-deleted notes or daily logs found to restore.")
				return nil
			}

			idxs, err := fuzzyfinder.FindMulti(items, func(i int) string {
				return items[i].FormatFuzzy()
			})
			if err != nil {
				if err == fuzzyfinder.ErrAbort {
					return nil
				}
				return err
			}

			for _, idx := range idxs {
				if err := restoreSingleItem(items[idx].ID); err != nil {
					return err
				}
			}

			return nil
		}

		for _, id := range args {
			if err := restoreSingleItem(id); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	restoreCmd.ValidArgsFunction = completeDeletedIDs
	rootCmd.AddCommand(restoreCmd)
}
