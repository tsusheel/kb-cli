package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/utils"
)

var historyLimit int

var historyCmd = &cobra.Command{
	Use:     "history [id]",
	Aliases: []string{"audit", "revisions", "changelog"},
	Short:   "View audit and version history for notes and daily logs",
	Long: `View the modification history, diffs, and audit trail.

Examples:
  # View recent global audit activity stream:
  kb history
  kb audit

  # View modification history for a specific note:
  kb history d8d1e07

  # Inspect a specific revision snapshot:
  kb history diff d8d1e07 a1b2c3d

  # Revert a note to a previous revision:
  kb history revert d8d1e07 a1b2c3d`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// Global recent audit history
			entries, err := db.GetRecentAuditHistory(historyLimit)
			if err != nil {
				return fmt.Errorf("failed fetching recent audit history: %w", err)
			}
			utils.RenderAuditTable(entries, "=== Recent Knowledge Base Activity ===", nil)
			return nil
		}

		id := args[0]

		// 1. Try finding as Note
		if n, err := db.GetNote(id); err == nil {
			entries, err := db.GetAuditHistory("note", n.ID, historyLimit)
			if err != nil {
				return fmt.Errorf("failed fetching history for note [%s]: %w", utils.ShortID(n.ID), err)
			}
			displayTitle := n.Note
			if displayTitle == "" {
				displayTitle = "<Untitled>"
			}
			title := fmt.Sprintf("=== History for Note [%s] %q ===", utils.ShortID(n.ID), displayTitle)
			utils.RenderAuditTable(entries, title, nil)
			return nil
		}

		// 2. Try finding as Daily Log
		if l, err := db.GetDailyLog(id); err == nil {
			entries, err := db.GetAuditHistory("daily_log", l.ID, historyLimit)
			if err != nil {
				return fmt.Errorf("failed fetching history for log [%s]: %w", utils.ShortID(l.ID), err)
			}
			logContent := l.Content
			if len(logContent) > 40 {
				logContent = logContent[:37] + "..."
			}
			title := fmt.Sprintf("=== History for Daily Log [%s] %q ===", utils.ShortID(l.ID), logContent)
			utils.RenderAuditTable(entries, title, nil)
			return nil
		}

		return fmt.Errorf("no note or daily log found matching ID %q", id)
	},
}

var historyDiffCmd = &cobra.Command{
	Use:     "diff <id> [rev_id]",
	Short:   "Inspect details and snapshot of an audit revision",
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		var revID string
		if len(args) == 2 {
			revID = args[1]
		}

		if revID != "" {
			entry, err := db.GetAuditEntry(revID)
			if err != nil {
				return fmt.Errorf("revision %q not found: %w", revID, err)
			}
			utils.RenderAuditDiffCard(entry, nil)
			return nil
		}

		// If rev_id is omitted, get latest revision for entity
		var entityType string
		var entityID string
		if n, err := db.GetNote(id); err == nil {
			entityType = "note"
			entityID = n.ID
		} else if l, err := db.GetDailyLog(id); err == nil {
			entityType = "daily_log"
			entityID = l.ID
		} else {
			return fmt.Errorf("no note or log found matching %q", id)
		}

		entries, err := db.GetAuditHistory(entityType, entityID, 1)
		if err != nil || len(entries) == 0 {
			return fmt.Errorf("no audit revisions found for %s [%s]", entityType, utils.ShortID(entityID))
		}

		utils.RenderAuditDiffCard(&entries[0], nil)
		return nil
	},
}

var historyRevertCmd = &cobra.Command{
	Use:     "revert <id> <rev_id>",
	Short:   "Revert a note to a previous point-in-time snapshot",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		revID := args[1]

		n, err := db.GetNote(id)
		if err != nil {
			return fmt.Errorf("note %q not found: %w", id, err)
		}

		entry, err := db.GetAuditEntry(revID)
		if err != nil {
			return fmt.Errorf("revision %q not found: %w", revID, err)
		}

		if entry.EntityType != "note" || !strings.HasPrefix(entry.EntityID, n.ID) {
			return fmt.Errorf("revision [%s] belongs to %s [%s], not note [%s]", utils.ShortID(entry.ID), entry.EntityType, utils.ShortID(entry.EntityID), utils.ShortID(n.ID))
		}

		if strings.TrimSpace(entry.SnapshotJSON) == "" {
			return fmt.Errorf("revision [%s] does not contain a recoverable snapshot", utils.ShortID(entry.ID))
		}

		reverted, err := db.RevertNoteToSnapshot(n.ID, entry.SnapshotJSON)
		if err != nil {
			return fmt.Errorf("failed reverting note: %w", err)
		}

		fmt.Printf("✔ Reverted note [%s] to revision [%s] (%s)\n", utils.ShortID(n.ID), utils.ShortID(entry.ID), entry.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("  Title:  %s\n", reverted.Note)
		fmt.Printf("  Status: %s | Type: %s\n", reverted.Status, reverted.Type)
		return nil
	},
}

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "limit", "l", 30, "Maximum number of audit records to display")
	historyCmd.ValidArgsFunction = completeNoteIDs

	historyDiffCmd.ValidArgsFunction = completeAuditIDs
	historyRevertCmd.ValidArgsFunction = completeAuditIDs

	historyCmd.AddCommand(historyDiffCmd)
	historyCmd.AddCommand(historyRevertCmd)

	rootCmd.AddCommand(historyCmd)
}

