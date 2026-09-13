package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/sync"
	"github.com/tsusheel/kb-cli/utils"
)

func buildPostgresConnStr() (string, error) {
	rawURL := utils.GetPostgresURL()
	if rawURL == "" {
		return "", fmt.Errorf("remote PostgreSQL URL is not configured. Run 'kb config setup' or 'kb config set remote.postgres_url <url>'")
	}

	// Parse URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid postgres connection URL: %w", err)
	}

	// Check if password already in URL, otherwise retrieve from OS Keyring
	if parsed.User != nil {
		_, hasPass := parsed.User.Password()
		if !hasPass {
			// Retrieve password from OS Keyring or env var
			secretPass, err := utils.GetSecret("postgres_password")
			if err == nil && secretPass != "" {
				parsed.User = url.UserPassword(parsed.User.Username(), secretPass)
			}
		}
	}

	return parsed.String(), nil
}

func getPostgresClient() (*sync.PostgresClient, error) {
	if !utils.IsRemoteEnabled() {
		return nil, fmt.Errorf("remote synchronization is disabled in config (remote.enabled = false)\nTo enable, run: kb config set remote.enabled true")
	}

	connStr, err := buildPostgresConnStr()
	if err != nil {
		return nil, err
	}

	return sync.NewPostgresClient(connStr)
}

func printSyncSummary(title string, stats *sync.SyncStats) {
	fmt.Printf("\n=== %s (%s) ===\n", title, stats.Duration.Round(time.Millisecond))
	table := tablewriter.NewTable(
		os.Stdout,
		tablewriter.WithHeader([]string{"ENTITY", "PUSHED", "PULLED"}),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleRounded),
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows:    tw.On,
					BetweenColumns: tw.On,
					ShowHeader:     tw.On,
				},
			},
		}),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	table.Append([]string{"Notes", fmt.Sprintf("%d", stats.NotesPushed), fmt.Sprintf("%d", stats.NotesPulled)})
	table.Append([]string{"Tags", fmt.Sprintf("%d", stats.TagsPushed), fmt.Sprintf("%d", stats.TagsPulled)})
	table.Append([]string{"Links", fmt.Sprintf("%d", stats.LinksPushed), fmt.Sprintf("%d", stats.LinksPulled)})
	table.Append([]string{"Daily Logs", fmt.Sprintf("%d", stats.LogsPushed), fmt.Sprintf("%d", stats.LogsPulled)})
	table.Render()
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize local SQLite knowledge base with remote PostgreSQL database (two-way)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
			return cmd.Help()
		}

		client, err := getPostgresClient()
		if err != nil {
			return err
		}
		defer client.Close()

		fmt.Println("Synchronizing with remote PostgreSQL database...")
		stats, err := client.TwoWaySync()
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		printSyncSummary("Sync Complete", stats)
		return nil
	},
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push local modifications to remote PostgreSQL database",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getPostgresClient()
		if err != nil {
			return err
		}
		defer client.Close()

		// Ensure remote schema exists
		if err := client.InitRemoteSchema(); err != nil {
			return err
		}

		lastSync, err := db.GetLastSyncedAt("postgres")
		if err != nil {
			return err
		}

		start := time.Now()
		fmt.Println("Pushing local changes to PostgreSQL...")
		stats, err := client.PushLocalChanges(lastSync)
		if err != nil {
			return fmt.Errorf("push failed: %w", err)
		}
		stats.Duration = time.Since(start)

		printSyncSummary("Push Complete", stats)
		return nil
	},
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull remote modifications from PostgreSQL into local database",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getPostgresClient()
		if err != nil {
			return err
		}
		defer client.Close()

		// Ensure remote schema exists
		if err := client.InitRemoteSchema(); err != nil {
			return err
		}

		lastSync, err := db.GetLastSyncedAt("postgres")
		if err != nil {
			return err
		}

		start := time.Now()
		fmt.Println("Pulling remote changes from PostgreSQL...")
		stats, err := client.PullRemoteChanges(lastSync)
		if err != nil {
			return fmt.Errorf("pull failed: %w", err)
		}
		stats.Duration = time.Since(start)

		printSyncSummary("Pull Complete", stats)
		return nil
	},
}

var syncStatusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"info"},
	Short:   "Display remote synchronization and connection status",
	RunE: func(cmd *cobra.Command, args []string) error {
		rawURL := utils.GetPostgresURL()
		hasPass := utils.HasSecret("postgres_password") || strings.Contains(rawURL, ":")
		lastSync, _ := db.GetLastSyncedAt("postgres")
		displayURL := utils.MaskURL(rawURL)
		enabled := utils.IsRemoteEnabled()

		notesCount, logsCount, _ := db.GetUnsyncedCounts(lastSync)

		info := utils.SyncStatusInfo{
			Enabled:       enabled,
			Provider:      "PostgreSQL",
			DatabaseURL:   displayURL,
			HasPassword:   hasPass,
			LastSyncedAt:  lastSync,
			UnsyncedNotes: notesCount,
			UnsyncedLogs:  logsCount,
		}

		utils.RenderSyncStatusTable(info, os.Stdout)

		if rawURL == "" {
			fmt.Println("\nTip: Run 'kb config setup' to configure remote PostgreSQL synchronization.")
		} else if !enabled {
			fmt.Println("\nTip: Run 'kb config set remote.enabled true' to enable synchronization.")
		}
		return nil
	},
}

var syncTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test connection to remote PostgreSQL database",
	RunE: func(cmd *cobra.Command, args []string) error {
		connStr, err := buildPostgresConnStr()
		if err != nil {
			return err
		}

		client, err := sync.NewPostgresClient(connStr)
		if err != nil {
			return err
		}
		defer client.Close()

		fmt.Print("Testing connection to PostgreSQL database... ")
		if err := client.TestConnection(); err != nil {
			fmt.Println("❌ FAILED")
			return err
		}
		fmt.Println("✔ Connection SUCCESSFUL")
		if !utils.IsRemoteEnabled() {
			fmt.Println("ℹ Note: Remote synchronization is currently disabled in config (remote.enabled = false).")
		}
		return nil
	},
}

func init() {
	syncCmd.AddCommand(syncPushCmd)
	syncCmd.AddCommand(syncPullCmd)
	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncTestCmd)
	rootCmd.AddCommand(syncCmd)
}

