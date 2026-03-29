package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
)

var linkTypeArg string

var linkCmd = &cobra.Command{
	Use:   "link <from_id> <to_id>",
	Short: "Link two notes together",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fromID := args[0]
		toID := args[1]

		if err := db.AddLink(fromID, toID, models.LinkType(linkTypeArg)); err != nil {
			return fmt.Errorf("failed to link notes: %w", err)
		}

		fmt.Printf("Successfully linked [%s] --> [%s] as '%s'\n", fromID, toID, linkTypeArg)
		return nil
	},
}

func init() {
	linkCmd.Flags().StringVarP(&linkTypeArg, "type", "t", string(models.RelatedTo), "Type of link (e.g., related_to, part_of, depends_on)")
	rootCmd.AddCommand(linkCmd)
}
