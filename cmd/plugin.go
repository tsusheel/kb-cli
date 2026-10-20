package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/plugins"
	"github.com/tsusheel/kb-cli/plugins/loader"
)

var pluginCmd = &cobra.Command{
	Use:     "plugin",
	Aliases: []string{"plugins"},
	Short:   "Manage external and built-in plugins",
	Long: `Inspect, install, enable, disable, and verify plugins for kb-cli.

Examples:
  # List all installed plugins:
  kb plugin list

  # Install an external plugin binary:
  kb plugin install ./bin/kb-plugin-gym

  # Enable a plugin:
  kb plugin enable gym

  # Disable a plugin:
  kb plugin disable gym

  # Grant or revoke a capability:
  kb plugin grant gym storage.entries
  kb plugin revoke gym secrets

  # Run diagnostic health check:
  kb plugin doctor gym`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all discovered and installed plugins",
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := db.ListPluginRecords()
		if err != nil {
			return fmt.Errorf("failed listing plugins: %w", err)
		}

		if len(records) == 0 {
			fmt.Println("No plugins installed.")
			return nil
		}

		table := tablewriter.NewTable(
			os.Stdout,
			tablewriter.WithHeader([]string{"PLUGIN ID", "VERSION", "STATUS", "SOURCE", "GRANTED CAPABILITIES"}),
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
		)

		for _, p := range records {
			status := "disabled"
			if p.Enabled {
				status = "enabled"
			}

			caps, _ := db.GetPluginCapabilities(p.PluginID)
			capsStr := strings.Join(caps, ", ")
			if capsStr == "" {
				capsStr = "(none)"
			}

			table.Append([]string{
				p.PluginID,
				p.Version,
				status,
				p.Source,
				capsStr,
			})
		}

		table.Render()
		return nil
	},
}

var pluginEnableCmd = &cobra.Command{
	Use:   "enable [plugin_id]",
	Short: "Enable an installed plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginID := args[0]
		mgr := plugins.GetManager()
		if mgr == nil {
			mgr = plugins.InitManager(rootCmd, nil, nil)
		}

		if err := mgr.EnablePlugin(context.Background(), pluginID); err != nil {
			return fmt.Errorf("failed enabling plugin %q: %w", pluginID, err)
		}

		fmt.Printf("✔ Successfully enabled plugin %q\n", pluginID)
		return nil
	},
}

var pluginDisableCmd = &cobra.Command{
	Use:   "disable [plugin_id]",
	Short: "Disable an active plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginID := args[0]
		mgr := plugins.GetManager()
		if mgr == nil {
			mgr = plugins.InitManager(rootCmd, nil, nil)
		}

		if err := mgr.DisablePlugin(context.Background(), pluginID); err != nil {
			return fmt.Errorf("failed disabling plugin %q: %w", pluginID, err)
		}

		fmt.Printf("✔ Successfully disabled plugin %q (data remains preserved)\n", pluginID)
		return nil
	},
}

var pluginGrantCmd = &cobra.Command{
	Use:   "grant [plugin_id] [capability]",
	Short: "Grant a capability permission to a plugin",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginID := args[0]
		capability := args[1]

		if err := plugins.GrantCapability(pluginID, capability); err != nil {
			return fmt.Errorf("failed granting capability: %w", err)
		}

		fmt.Printf("✔ Granted capability %q to plugin %q\n", capability, pluginID)
		return nil
	},
}

var pluginRevokeCmd = &cobra.Command{
	Use:   "revoke [plugin_id] [capability]",
	Short: "Revoke a capability permission from a plugin",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginID := args[0]
		capability := args[1]

		if err := plugins.RevokeCapability(pluginID, capability); err != nil {
			return fmt.Errorf("failed revoking capability: %w", err)
		}

		fmt.Printf("✔ Revoked capability %q from plugin %q\n", capability, pluginID)
		return nil
	},
}

var pluginDoctorCmd = &cobra.Command{
	Use:   "doctor [plugin_id]",
	Short: "Run diagnostics on a plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginID := args[0]
		rec, err := db.GetPluginRecord(pluginID)
		if err != nil {
			return fmt.Errorf("plugin %q is not installed in database", pluginID)
		}

		fmt.Printf("=== Diagnostic Report for Plugin [%s] ===\n", pluginID)
		fmt.Printf("  Version:        %s\n", rec.Version)
		fmt.Printf("  API Version:    %s\n", rec.APIVersion)
		fmt.Printf("  Schema Version: %d\n", rec.SchemaVersion)
		fmt.Printf("  Source:         %s\n", rec.Source)
		fmt.Printf("  Enabled:        %v\n", rec.Enabled)

		// Check migrations
		migrations, err := db.GetPluginMigrations(pluginID)
		if err != nil {
			fmt.Printf("  Migrations:     Error reading: %v\n", err)
		} else {
			fmt.Printf("  Migrations:     %d applied\n", len(migrations))
			for _, m := range migrations {
				fmt.Printf("    - v%d (checksum: %s, applied: %s)\n", m.Version, m.Checksum[:8], m.AppliedAt.Format("2006-01-02 15:04"))
			}
		}

		// Check capabilities
		caps, err := db.GetPluginCapabilities(pluginID)
		if err != nil {
			fmt.Printf("  Capabilities:   Error reading: %v\n", err)
		} else {
			fmt.Printf("  Capabilities:   %d granted (%s)\n", len(caps), strings.Join(caps, ", "))
		}

		return nil
	},
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install [executable_path]",
	Short: "Install an external plugin binary",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		fmt.Printf("Inspecting plugin binary at %s...\n", path)
		proc, err := loader.NewProcessPlugin(path)
		if err != nil {
			return fmt.Errorf("failed inspecting plugin executable: %w", err)
		}
		defer func() {
			_ = proc.Stop(context.Background())
		}()

		manifest := proc.Manifest()
		fmt.Printf("✔ Valid plugin manifest detected:\n")
		fmt.Printf("  ID:           %s\n", manifest.ID)
		fmt.Printf("  Name:         %s\n", manifest.Name)
		fmt.Printf("  Version:      %s\n", manifest.Version)
		fmt.Printf("  Capabilities: %s\n", strings.Join(manifest.Capabilities, ", "))

		rec := &models.PluginRecord{
			PluginID:      manifest.ID,
			Version:       manifest.Version,
			APIVersion:    manifest.APIVersion,
			SchemaVersion: manifest.SchemaVersion,
			Enabled:       false,
			Source:        path,
			ManifestJSON:  fmt.Sprintf(`{"id":%q,"name":%q,"version":%q}`, manifest.ID, manifest.Name, manifest.Version),
		}

		if err := db.RegisterPluginRecord(rec); err != nil {
			return fmt.Errorf("failed registering plugin in database: %w", err)
		}

		// Automatically grant declared capabilities upon explicit install
		for _, cap := range manifest.Capabilities {
			_ = plugins.GrantCapability(manifest.ID, cap)
		}

		fmt.Printf("\n✔ Plugin %q installed successfully! Run 'kb plugin enable %s' to activate.\n", manifest.ID, manifest.ID)
		return nil
	},
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
	pluginCmd.AddCommand(pluginGrantCmd)
	pluginCmd.AddCommand(pluginRevokeCmd)
	pluginCmd.AddCommand(pluginDoctorCmd)

	rootCmd.AddCommand(pluginCmd)
}
