package plugins

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/db"
	"github.com/tsusheel/kb-cli/models"
	"github.com/tsusheel/kb-cli/plugins/api"
	"github.com/tsusheel/kb-cli/sync"
)

type mockPlugin struct {
	started bool
	stopped bool
}

func (m *mockPlugin) Manifest() api.Manifest {
	return api.Manifest{
		ID:             "mock_plugin",
		Name:           "Mock Plugin",
		Version:        "1.0.0",
		APIVersion:     "1",
		SchemaVersion:  1,
		MinCoreVersion: "1.0.0",
		Capabilities: []string{
			api.CapStorageEntries,
			api.CapCLICommands,
			api.CapMCPTools,
		},
	}
}

func (m *mockPlugin) Register(ctx *api.PluginContext) error {
	// 1. Test entry creation
	err := ctx.Store.Create(context.Background(), api.Entry{
		EntityType: "mock_item",
		SearchText: "mock search test",
		Data:       []byte(`{"title":"hello"}`),
	})
	if err != nil {
		return err
	}

	// 2. Test CLI command registration
	ctx.Commands.AddCommand(&cobra.Command{
		Use:   "mock-cmd",
		Short: "Mock command",
		Run:   func(cmd *cobra.Command, args []string) {},
	})

	// 3. Test MCP tool registration
	_ = ctx.MCP.RegisterTool(api.MCPToolDef{
		Name:        "mock_echo",
		Description: "Echoes input back",
		Handler: func(ctx context.Context, args []byte) (any, error) {
			return string(args), nil
		},
	})

	return nil
}

func (m *mockPlugin) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *mockPlugin) Stop(ctx context.Context) error {
	m.stopped = true
	return nil
}

func TestPluginLifecycleAndCapabilities(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_plugins.db")

	if err := db.InitDB(dbPath); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.CloseDB()
	_ = db.InitSchema()

	rootCmd := &cobra.Command{Use: "kb"}
	syncReg := sync.NewAdapterRegistry()
	mgr := InitManager(rootCmd, nil, syncReg)

	p := &mockPlugin{}

	// Install plugin
	if err := mgr.InstallBuiltin(p, false); err != nil {
		t.Fatalf("InstallBuiltin failed: %v", err)
	}

	// Enable plugin
	if err := mgr.EnablePlugin(context.Background(), "mock_plugin"); err != nil {
		t.Fatalf("EnablePlugin failed: %v", err)
	}
	if !p.started {
		t.Errorf("expected plugin to be started")
	}

	// Verify command was attached to rootCmd
	foundCmd := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "mock-cmd" {
			foundCmd = true
			break
		}
	}
	if !foundCmd {
		t.Errorf("expected mock-cmd to be attached to rootCmd")
	}

	// Verify MCP tool dispatch
	res, err := DispatchMCPTool(context.Background(), "mock_echo", []byte("hello world"))
	if err != nil || res != "hello world" {
		t.Errorf("unexpected MCP tool result: %v, err: %v", res, err)
	}

	// Verify entry was written into plugin_entries
	entries, err := db.QueryEntries(models.EntryFilter{PluginID: "mock_plugin"})
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 entry for mock_plugin, got %d, err: %v", len(entries), err)
	}

	// Disable plugin
	if err := mgr.DisablePlugin(context.Background(), "mock_plugin"); err != nil {
		t.Fatalf("DisablePlugin failed: %v", err)
	}
	if !p.stopped {
		t.Errorf("expected plugin to be stopped")
	}
}
