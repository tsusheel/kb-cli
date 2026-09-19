package cmd

import (
	"fmt"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
)

var deleteReason string

func deleteSingleItem(id string, reason string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id cannot be empty")
	}

	if reason == "" {
		reason = "deleted by user"
	}

	// 1. Try deleting as Note
	if n, err := db.GetNote(id); err == nil {
		if err := db.SoftDeleteNote(n.ID, reason); err != nil {
			return fmt.Errorf("failed deleting note [%s]: %w", n.ID[:7], err)
		}
		fmt.Printf("✔ Soft-deleted note [%s] %s\n", n.ID[:7], n.Note)
		return nil
	}

	// 2. Try deleting as Daily Log
	if l, err := db.GetDailyLog(id); err == nil {
		if err := db.SoftDeleteDailyLog(l.ID, reason); err != nil {
			return fmt.Errorf("failed deleting daily log [%s]: %w", l.ID[:7], err)
		}
		logContent := l.Content
		if len(logContent) > 40 {
			logContent = logContent[:37] + "..."
		}
		fmt.Printf("✔ Soft-deleted daily log [%s] %s\n", l.ID[:7], logContent)
		return nil
	}

	return fmt.Errorf("no note or daily log found matching ID %q", id)
}

var deleteCmd = &cobra.Command{
	Use:     "delete [id...]",
	Aliases: []string{"rm", "del", "remove"},
	Short:   "Soft-delete notes or daily logs (fuzzy-select if ID is omitted)",
	Long: `Soft-delete notes or daily logs by ID.

When run without arguments, launches an interactive multi-selection fuzzy finder containing all active notes and daily logs.
All deletions are non-destructive and preserve an audit trail (deleted_at timestamp and attribution reason).

Examples:
  # Delete a note by ID:
  kb delete 3aba340
  kb rm 3aba340

  # Delete with custom reason:
  kb delete 3aba340 --reason "superseded by new spec"

  # Fuzzy-find and select notes/logs to delete (Tab to multi-select, Enter to confirm):
  kb delete
  kb rm

  # Delete a daily log:
  kb delete 6a178a8`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			items, err := GetActiveCLIItems()
			if err != nil {
				return err
			}

			if len(items) == 0 {
				fmt.Println("No active notes or daily logs found to delete.")
				return nil
			}

			idxs, err := fuzzyfinder.FindMulti(items, func(i int) string {
				return items[i].FormatFuzzy()
			}, fuzzyfinder.WithPreviewWindow(func(i int, width, height int) string {
				if i < 0 || i >= len(items) {
					return ""
				}
				return items[i].RenderPreview()
			}))
			if err != nil {
				if err == fuzzyfinder.ErrAbort {
					return nil
				}
				return err
			}

			for _, idx := range idxs {
				if err := deleteSingleItem(items[idx].ID, deleteReason); err != nil {
					return err
				}
			}

			return nil
		}

		for _, id := range args {
			if err := deleteSingleItem(id, deleteReason); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	deleteCmd.Flags().StringVarP(&deleteReason, "reason", "r", "deleted by user", "Attribution reason for soft delete")
	deleteCmd.ValidArgsFunction = completeDeletableIDs
	rootCmd.AddCommand(deleteCmd)
}
