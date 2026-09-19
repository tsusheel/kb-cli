package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/utils"
)

var statsCmd = &cobra.Command{
	Use:     "stats",
	Aliases: []string{"summary", "dashboard", "info"},
	Short:   "Display knowledge base metrics, activity streaks, and graph distribution",
	Long: `Display high-level statistics about your knowledge base:
- Total active and soft-deleted notes
- Breakdown by type, status, and area
- Daily logging volume and consecutive activity streaks
- Top tags and most connected hub notes

Examples:
  kb stats
  kb summary`,
	RunE: func(cmd *cobra.Command, args []string) error {
		stats, err := db.GetKnowledgeBaseStats()
		if err != nil {
			return fmt.Errorf("failed fetching stats: %w", err)
		}

		fmt.Println("=== Knowledge Base Analytics Dashboard ===")
		utils.RenderStatsDashboard(stats, os.Stdout)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
