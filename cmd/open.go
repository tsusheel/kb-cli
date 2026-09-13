package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/utils"
)

func displayItem(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id cannot be empty")
	}

	// 1. Try opening as Note
	if n, err := db.GetNote(id); err == nil {
		tags, _ := db.GetTagsForNote(n.ID)
		links, _ := db.GetLinksForNote(n.ID)
		utils.RenderNoteDetail(n, tags, links, os.Stdout)
		return nil
	}

	// 2. Try opening as Daily Log
	if l, err := db.GetDailyLog(id); err == nil {
		var promotedNote *models.Note
		if l.NoteID != "" {
			promotedNote, _ = db.GetNote(l.NoteID)
		}
		utils.RenderDailyLogDetail(l, promotedNote, os.Stdout)
		return nil
	}

	return fmt.Errorf("no note or daily log found matching ID %q", id)
}

var openCmd = &cobra.Command{
	Use:     "open [id]",
	Aliases: []string{"view"},
	Short:   "View note or daily log details (fuzzy-select if ID is omitted)",
	Long: `View the complete details and content of a note or daily log.

When run without arguments, launches an interactive fuzzy finder showing all active notes and daily micro-logs.

Examples:
  # Open note by ID:
  kb open 75f4c33
  kb view 75f4c33

  # Open daily log by ID:
  kb open 6fb3bab

  # Interactive fuzzy finder across notes and daily logs:
  kb open`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return displayItem(args[0])
		}

		var items []CLIItem

		// 1. Active Notes
		notes, err := db.ListNotes("")
		if err != nil {
			return err
		}
		for _, n := range notes {
			displayTitle := n.Note
			if displayTitle == "" {
				displayTitle = "<Untitled>"
			}
			items = append(items, CLIItem{
				ID:        n.ID,
				Type:      string(n.Type),
				Display:   displayTitle,
				Timestamp: n.UpdatedAt.Format("2006-01-02 15:04"),
				IsLog:     false,
			})
		}

		// 2. Active Daily Logs
		logs, err := db.GetDailyLogsFilter(nil, nil, true, false)
		if err != nil {
			return err
		}
		for _, l := range logs {
			content := l.Content
			if len(content) > 60 {
				content = content[:57] + "..."
			}
			logType := "log"
			if l.NoteID != "" {
				logType = "log:promoted"
			}
			items = append(items, CLIItem{
				ID:        l.ID,
				Type:      logType,
				Display:   content,
				Timestamp: l.CreatedAt.Format("2006-01-02 15:04"),
				IsLog:     true,
			})
		}

		if len(items) == 0 {
			fmt.Println("No active notes or daily logs found.")
			return nil
		}

		idx, err := fuzzyfinder.Find(items, func(i int) string {
			return items[i].FormatFuzzy()
		})
		if err != nil {
			if err == fuzzyfinder.ErrAbort {
				return nil
			}
			return err
		}

		return displayItem(items[idx].ID)
	},
}

func init() {
	openCmd.ValidArgsFunction = completeDeletableIDs
	rootCmd.AddCommand(openCmd)
}
