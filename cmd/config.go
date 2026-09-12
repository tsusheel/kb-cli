package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tsusheel/kb-cli/sync"
	"github.com/tsusheel/kb-cli/utils"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration settings and secure OS keyring secrets",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a public configuration option in config.yaml",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		val := args[1]

		// Check if user is attempting to save sensitive credentials in plaintext
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "key") || strings.Contains(lowerKey, "pass") || strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "token") {
			fmt.Printf("⚠️  Warning: %q looks like a secret or password.\n", key)
			fmt.Printf("   To keep your credentials secure, use: kb config set-secret %s\n\n", key)
		}

		viper.Set(key, val)
		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			home, _ := os.UserHomeDir()
			configFile = home + "/.config/kb/config.yaml"
		}

		if err := viper.WriteConfig(); err != nil {
			if err := viper.WriteConfigAs(configFile); err != nil {
				return fmt.Errorf("failed to write config file %s: %w", configFile, err)
			}
		}

		fmt.Printf("✔ Configuration updated: %s = %s\n", key, val)
		return nil
	},
}

var configSetSecretCmd = &cobra.Command{
	Use:   "set-secret <key>",
	Short: "Securely store a secret or password in the OS Credential Manager (masked input)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		secret, err := utils.PromptSecret(fmt.Sprintf("Enter secret for %q (input will be hidden): ", key))
		if err != nil {
			return err
		}

		if secret == "" {
			return fmt.Errorf("cannot store an empty secret")
		}

		if err := utils.SetSecret(key, secret); err != nil {
			return fmt.Errorf("failed to store secret in OS Keyring: %w", err)
		}

		fmt.Printf("✔ Secret %q securely saved in OS Credential Manager.\n", key)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		// Check if it's a secret in OS Keyring
		if utils.HasSecret(key) {
			fmt.Printf("%s: [STORED SECURELY IN OS KEYRING]\n", key)
			return nil
		}

		val := viper.Get(key)
		if val == nil {
			fmt.Printf("%s is not set.\n", key)
			return nil
		}

		fmt.Printf("%s = %v\n", key, val)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "show"},
	Short:   "List all configuration settings and secret statuses",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "KEY\tVALUE\tSOURCE")
		fmt.Fprintln(w, "---\t-----\t------")

		// List Viper settings
		keys := viper.AllKeys()
		for _, k := range keys {
			val := viper.GetString(k)
			// Mask passwords in URLs if any
			if strings.Contains(k, "url") || strings.Contains(k, "conn") {
				val = utils.MaskURL(val)
			}
			fmt.Fprintf(w, "%s\t%s\tconfig.yaml\n", k, val)
		}

		// List Known Secrets Status
		knownSecrets := []string{"postgres_password", "db_password", "remote_db_password"}
		for _, s := range knownSecrets {
			if utils.HasSecret(s) {
				fmt.Fprintf(w, "%s\t[ENCRYPTED / SECURE]\tOS Keyring\n", s)
			}
		}

		w.Flush()
		return nil
	},
}

var configSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for remote PostgreSQL database synchronization",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("=== PostgreSQL Remote Sync Setup ===")
		fmt.Println("This wizard will configure your remote PostgreSQL database for encrypted local-first sync.")
		fmt.Println()

		// 1. PostgreSQL Connection URL
		currentURL := utils.GetPostgresURL()
		if currentURL != "" {
			fmt.Printf("PostgreSQL Connection URL [%s]: ", currentURL)
		} else {
			fmt.Print("PostgreSQL Connection URL (e.g. postgres://username@localhost:5432/dbname?sslmode=disable): ")
		}

		inputURL, _ := reader.ReadString('\n')
		inputURL = strings.TrimSpace(inputURL)
		if inputURL != "" {
			currentURL = inputURL
		}
		if currentURL == "" {
			return fmt.Errorf("postgres connection URL cannot be empty")
		}

		parsed, err := url.Parse(currentURL)
		if err != nil {
			return fmt.Errorf("invalid postgres URL: %w", err)
		}

		// 2. PostgreSQL Password (Masked)
		var activePass string
		if parsed.User != nil {
			if pass, hasPass := parsed.User.Password(); hasPass {
				activePass = pass
			}
		}

		if activePass == "" {
			fmt.Println()
			passPrompt := "Enter PostgreSQL Password (input hidden, will be saved to OS Keyring): "
			if utils.HasSecret("postgres_password") {
				passPrompt = "Enter PostgreSQL Password (leave empty to keep existing password): "
			}

			inputPass, err := utils.PromptSecret(passPrompt)
			if err != nil {
				return err
			}

			if inputPass != "" {
				activePass = inputPass
			} else {
				activePass, _ = utils.GetSecret("postgres_password")
			}
		}

		// Clean URL to store in config (without password)
		configURL := currentURL
		if parsed.User != nil {
			// Strip password from config.yaml URL
			parsed.User = url.User(parsed.User.Username())
			configURL = parsed.String()
		}

		// Build full connection string with password for testing
		testURL := configURL
		if activePass != "" && parsed.User != nil {
			parsed.User = url.UserPassword(parsed.User.Username(), activePass)
			testURL = parsed.String()
		}

		// 3. Test Connection
		fmt.Print("\nTesting connection to PostgreSQL... ")
		client, err := sync.NewPostgresClient(testURL)
		if err != nil {
			return err
		}
		defer client.Close()

		if err := client.TestConnection(); err != nil {
			fmt.Println("❌ FAILED")
			fmt.Printf("Error: %v\n\n", err)
			return fmt.Errorf("connection verification failed")
		}

		// Initialize remote tables automatically
		if err := client.InitRemoteSchema(); err != nil {
			fmt.Println("❌ FAILED (Schema Init)")
			return fmt.Errorf("failed creating remote tables: %w", err)
		}

		fmt.Println("✔ SUCCESS")

		// 4. Save Config
		viper.Set("remote.enabled", true)
		viper.Set("remote.provider", "postgres")
		viper.Set("remote.postgres_url", configURL)
		if err := viper.WriteConfig(); err != nil {
			configFile := viper.ConfigFileUsed()
			_ = viper.WriteConfigAs(configFile)
		}

		if activePass != "" {
			if err := utils.SetSecret("postgres_password", activePass); err != nil {
				return fmt.Errorf("failed storing password in keyring: %w", err)
			}
		}

		fmt.Println("\n✔ Configuration successfully saved!")
		fmt.Println("  - Database URL (without password) saved to config.yaml")
		fmt.Println("  - Password securely encrypted in OS Credential Manager")
		fmt.Println("  - Remote tables automatically verified and created")
		fmt.Println("\nYou can now run 'kb sync' to synchronize your knowledge base!")
		return nil
	},
}

func init() {
	configSetCmd.ValidArgsFunction = completeConfigKeys
	configGetCmd.ValidArgsFunction = completeConfigKeys
	configSetSecretCmd.ValidArgsFunction = completeSecretKeys

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configSetSecretCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configSetupCmd)
	rootCmd.AddCommand(configCmd)
}
