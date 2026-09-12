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
	Short:   "Soft-delete a note or daily log (fuzzy-select if ID is omitted)",
	Long: `Soft-delete a note or daily log by ID.

All deletions are non-destructive and preserve an audit trail (deleted_at timestamp and attribution reason).

Examples:
  # Delete a note by ID:
  kb delete 3aba340
  kb rm 3aba340

  # Delete with custom reason:
  kb delete 3aba340 --reason "superseded by new spec"

  # Fuzzy-find and select a note to delete:
  kb delete

  # Delete a daily log:
  kb delete log 6a178a8`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// Fuzzy finder mode
			notes, err := db.ListNotes("")
			if err != nil {
				return err
			}
			if len(notes) == 0 {
				fmt.Println("No active notes found to delete.")
				return nil
			}

			idx, err := fuzzyfinder.Find(notes, func(i int) string {
				displayTitle := notes[i].Note
				if displayTitle == "" {
					displayTitle = "<Untitled>"
				}
				return fmt.Sprintf("[%s] %s (%s)", notes[i].ID[:7], displayTitle, notes[i].Type)
			})
			if err != nil {
				if err == fuzzyfinder.ErrAbort {
					return nil
				}
				return err
			}

			return deleteSingleItem(notes[idx].ID, deleteReason)
		}

		for _, id := range args {
			if err := deleteSingleItem(id, deleteReason); err != nil {
				return err
			}
		}

		return nil
	},
}

var deleteLogCmd = &cobra.Command{
	Use:     "log <id>",
	Short:   "Soft-delete a specific daily log by ID",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reason := deleteReason
		if reason == "" {
			reason = "deleted by user"
		}
		l, err := db.GetDailyLog(args[0])
		if err != nil {
			return err
		}
		if err := db.SoftDeleteDailyLog(l.ID, reason); err != nil {
			return fmt.Errorf("failed deleting daily log: %w", err)
		}
		logContent := l.Content
		if len(logContent) > 40 {
			logContent = logContent[:37] + "..."
		}
		fmt.Printf("✔ Soft-deleted daily log [%s] %s\n", l.ID[:7], logContent)
		return nil
	},
}

var deleteNoteCmd = &cobra.Command{
	Use:     "note <id>",
	Short:   "Soft-delete a specific note by ID",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reason := deleteReason
		if reason == "" {
			reason = "deleted by user"
		}
		n, err := db.GetNote(args[0])
		if err != nil {
			return err
		}
		if err := db.SoftDeleteNote(n.ID, reason); err != nil {
			return fmt.Errorf("failed deleting note: %w", err)
		}
		fmt.Printf("✔ Soft-deleted note [%s] %s\n", n.ID[:7], n.Note)
		return nil
	},
}

func init() {
	deleteCmd.Flags().StringVarP(&deleteReason, "reason", "r", "deleted by user", "Attribution reason for soft delete")
	deleteLogCmd.Flags().StringVarP(&deleteReason, "reason", "r", "deleted by user", "Attribution reason for soft delete")
	deleteNoteCmd.Flags().StringVarP(&deleteReason, "reason", "r", "deleted by user", "Attribution reason for soft delete")

	deleteCmd.ValidArgsFunction = completeNoteIDs
	deleteNoteCmd.ValidArgsFunction = completeNoteIDs
	deleteLogCmd.ValidArgsFunction = completeUnpromotedLogIDs

	deleteCmd.AddCommand(deleteLogCmd)
	deleteCmd.AddCommand(deleteNoteCmd)
	rootCmd.AddCommand(deleteCmd)
}
