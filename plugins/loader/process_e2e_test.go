package loader_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"

	"github.com/tsusheel/kb-cli/plugins/api"
	"github.com/tsusheel/kb-cli/plugins/loader"
)

// TestProcessPlugin_E2E builds and launches the dummy_plugin fixture binary,
// then exercises the full JSON-RPC lifecycle: handshake, register, command.execute,
// mcp.call, and stop.
func TestProcessPlugin_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E process plugin test in short mode")
	}

	// Build the dummy plugin fixture
	fixtureDir := filepath.Join("..", "..", "test", "fixtures", "dummy_plugin")
	binaryName := "kb-plugin-dummy"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, binaryName)

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = fixtureDir
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	out, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed building dummy plugin: %v\n%s", err, out)
	}

	// 1. Spawn the process plugin (implicitly performs handshake)
	proc, err := loader.NewProcessPlugin(binaryPath)
	if err != nil {
		t.Fatalf("NewProcessPlugin failed: %v", err)
	}
	defer func() {
		_ = proc.Stop(context.Background())
	}()

	// 2. Verify handshake returned a valid manifest
	manifest := proc.Manifest()
	if manifest.ID != "dummy" {
		t.Errorf("expected manifest ID 'dummy', got %q", manifest.ID)
	}
	if manifest.Name != "Dummy Test Plugin" {
		t.Errorf("expected manifest Name 'Dummy Test Plugin', got %q", manifest.Name)
	}
	if manifest.Version != "0.1.0" {
		t.Errorf("expected manifest Version '0.1.0', got %q", manifest.Version)
	}
	if manifest.APIVersion != "1" {
		t.Errorf("expected APIVersion '1', got %q", manifest.APIVersion)
	}
	if len(manifest.Capabilities) != 3 {
		t.Errorf("expected 3 capabilities, got %d: %v", len(manifest.Capabilities), manifest.Capabilities)
	}

	// 3. Test register RPC — calls "register" and captures declared commands/tools
	mockCommands := &testCommandRegistry{}
	mockMCP := &testMCPRegistry{}
	pCtx := &api.PluginContext{
		PluginID: "dummy",
		Commands: mockCommands,
		MCP:      mockMCP,
	}

	if err := proc.Register(pCtx); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if mockCommands.count != 1 {
		t.Errorf("expected 1 command registered, got %d", mockCommands.count)
	}
	if mockMCP.count != 1 {
		t.Errorf("expected 1 MCP tool registered, got %d", mockMCP.count)
	}

	// 4. Test start RPC
	if err := proc.Start(context.Background()); err != nil {
		t.Fatalf("Start RPC failed: %v", err)
	}

	t.Log("✔ E2E process plugin test passed: handshake, register, start all succeeded")
}

// testCommandRegistry is a mock that counts AddCommand calls.
type testCommandRegistry struct {
	count int
}

func (c *testCommandRegistry) AddCommand(cmd *cobra.Command) {
	c.count++
}

// testMCPRegistry is a mock that counts RegisterTool calls.
type testMCPRegistry struct {
	count int
	tools []api.MCPToolDef
}

func (m *testMCPRegistry) RegisterTool(tool api.MCPToolDef) error {
	m.count++
	m.tools = append(m.tools, tool)
	return nil
}

// TestProcessPlugin_ManifestCapabilities verifies that the manifest capabilities
// from the dummy plugin are valid known capability strings.
func TestProcessPlugin_ManifestCapabilities(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	fixtureDir := filepath.Join("..", "..", "test", "fixtures", "dummy_plugin")
	binaryName := "kb-plugin-dummy"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, binaryName)

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = fixtureDir
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	out, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed building dummy plugin: %v\n%s", err, out)
	}

	proc, err := loader.NewProcessPlugin(binaryPath)
	if err != nil {
		t.Fatalf("NewProcessPlugin failed: %v", err)
	}
	defer func() {
		_ = proc.Stop(context.Background())
	}()

	validCaps := map[string]bool{
		api.CapStorageEntries: true,
		api.CapStorageDomain:  true,
		api.CapCLICommands:    true,
		api.CapMCPTools:       true,
		api.CapHTTPRoutes:     true,
		api.CapAudit:          true,
		api.CapSecrets:        true,
		api.CapSync:           true,
		api.CapSearch:         true,
	}

	for _, cap := range proc.Manifest().Capabilities {
		if !validCaps[cap] {
			t.Errorf("unknown capability %q in manifest", cap)
		}
	}
}

// Ensure json import is used
var _ = json.Marshal
