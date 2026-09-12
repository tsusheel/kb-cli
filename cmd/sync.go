package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

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
	connStr, err := buildPostgresConnStr()
	if err != nil {
		return nil, err
	}

	return sync.NewPostgresClient(connStr)
}

func printSyncSummary(title string, stats *sync.SyncStats) {
	fmt.Printf("=== %s (%s) ===\n", title, stats.Duration.Round(time.Millisecond))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Entity\tPushed\tPulled\n")
	fmt.Fprintf(w, "------\t------\t------\n")
	fmt.Fprintf(w, "Notes\t%d\t%d\n", stats.NotesPushed, stats.NotesPulled)
	fmt.Fprintf(w, "Tags\t%d\t%d\n", stats.TagsPushed, stats.TagsPulled)
	fmt.Fprintf(w, "Links\t%d\t%d\n", stats.LinksPushed, stats.LinksPulled)
	fmt.Fprintf(w, "Daily Logs\t%d\t%d\n", stats.LogsPushed, stats.LogsPulled)
	w.Flush()
	fmt.Println("======================================")
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize local SQLite knowledge base with remote PostgreSQL database (two-way)",
	RunE: func(cmd *cobra.Command, args []string) error {
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
	Use:   "status",
	Short: "Display remote synchronization and connection status",
	RunE: func(cmd *cobra.Command, args []string) error {
		rawURL := utils.GetPostgresURL()
		hasPass := utils.HasSecret("postgres_password") || strings.Contains(rawURL, ":")
		lastSync, _ := db.GetLastSyncedAt("postgres")
		displayURL := utils.MaskURL(rawURL)

		fmt.Println("=== PostgreSQL Remote Sync Status ===")
		if displayURL != "" {
			fmt.Printf("Database URL: %s\n", displayURL)
		} else {
			fmt.Println("Database URL: [Not Configured]")
		}

		if hasPass {
			fmt.Println("Password:     [Configured in OS Keyring]")
		} else {
			fmt.Println("Password:     [Not Configured]")
		}

		if !lastSync.IsZero() {
			fmt.Printf("Last Synced:  %s\n", lastSync.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Println("Last Synced:  Never")
		}

		notesCount, logsCount, err := db.GetUnsyncedCounts(lastSync)
		if err == nil {
			fmt.Printf("Pending:      %d un-synced notes, %d un-synced logs\n", notesCount, logsCount)
		}

		fmt.Println("======================================")
		if rawURL == "" {
			fmt.Println("Tip: Run 'kb config setup' to configure remote PostgreSQL synchronization.")
		}
		return nil
	},
}

var syncTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test connection to remote PostgreSQL database",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getPostgresClient()
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
