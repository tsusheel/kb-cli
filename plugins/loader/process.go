package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/tsusheel/kb-cli/plugins/api"
)

// ProcessPlugin wraps an external native executable communicating over JSON-RPC 2.0 stdio.
type ProcessPlugin struct {
	mu             sync.Mutex
	executablePath string
	cmd            *exec.Cmd
	transport      *Transport
	manifest       api.Manifest
	stdin          io.WriteCloser
	stdout         io.ReadCloser
}

// NewProcessPlugin spawns the child process and performs the initial handshake.
func NewProcessPlugin(executablePath string) (*ProcessPlugin, error) {
	if _, err := os.Stat(executablePath); err != nil {
		return nil, fmt.Errorf("plugin executable not found at %s: %w", executablePath, err)
	}

	p := &ProcessPlugin{
		executablePath: executablePath,
	}

	if err := p.spawn(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *ProcessPlugin) spawn() error {
	cmd := exec.Command(p.executablePath)

	// Security sandbox: sanitize environment variables
	var safeEnv []string
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "KB_CONFIG=") ||
			strings.HasPrefix(env, "KB_DB_PATH=") ||
			strings.HasPrefix(env, "DATABASE_URL=") ||
			strings.HasPrefix(env, "POSTGRES_PASSWORD=") {
			continue
		}
		safeEnv = append(safeEnv, env)
	}
	cmd.Env = safeEnv

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed creating plugin stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("failed creating plugin stdout pipe: %w", err)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return fmt.Errorf("failed spawning plugin process: %w", err)
	}

	p.cmd = cmd
	p.stdin = stdin
	p.stdout = stdout
	p.transport = NewTransport(stdout, stdin)

	// Perform initial handshake to fetch manifest
	res, err := p.transport.Call("handshake", map[string]string{"core_version": "1.0.0", "api_version": "1"})
	if err != nil {
		_ = p.Stop(context.Background())
		return fmt.Errorf("plugin handshake failed: %w", err)
	}

	var m api.Manifest
	if err := json.Unmarshal(res, &m); err != nil {
		_ = p.Stop(context.Background())
		return fmt.Errorf("invalid manifest returned in handshake: %w", err)
	}

	p.manifest = m
	return nil
}

func (p *ProcessPlugin) Manifest() api.Manifest {
	return p.manifest
}

type CommandDef struct {
	Use   string `json:"use"`
	Short string `json:"short"`
	Long  string `json:"long"`
}

type MCPToolDefPayload struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	SchemaJSON  json.RawMessage `json:"schema_json"`
}

type RegisterResult struct {
	Commands []CommandDef        `json:"commands"`
	MCPTools []MCPToolDefPayload `json:"mcp_tools"`
}

func (p *ProcessPlugin) Register(ctx *api.PluginContext) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	res, err := p.transport.Call("register", map[string]string{"plugin_id": p.manifest.ID})
	if err != nil {
		return fmt.Errorf("plugin register rpc failed: %w", err)
	}

	var regResult RegisterResult
	if len(res) > 0 && string(res) != "null" {
		_ = json.Unmarshal(res, &regResult)
	}

	// Mount registered CLI commands
	for _, cDef := range regResult.Commands {
		cmdName := cDef.Use
		cmd := &cobra.Command{
			Use:   cmdName,
			Short: cDef.Short,
			Long:  cDef.Long,
			RunE: func(cmd *cobra.Command, args []string) error {
				params := map[string]any{
					"command": cmdName,
					"args":    args,
				}
				outputRes, err := p.transport.Call("command.execute", params)
				if err != nil {
					return err
				}
				var outStr string
				if err := json.Unmarshal(outputRes, &outStr); err == nil && outStr != "" {
					fmt.Print(outStr)
				}
				return nil
			},
		}
		ctx.Commands.AddCommand(cmd)
	}

	// Mount registered MCP tools
	for _, tDef := range regResult.MCPTools {
		toolName := tDef.Name
		_ = ctx.MCP.RegisterTool(api.MCPToolDef{
			Name:        toolName,
			Description: tDef.Description,
			SchemaJSON:  tDef.SchemaJSON,
			Handler: func(callCtx context.Context, args []byte) (any, error) {
				params := map[string]any{
					"tool": toolName,
					"args": json.RawMessage(args),
				}
				callRes, err := p.transport.Call("mcp.call", params)
				if err != nil {
					return nil, err
				}
				return json.RawMessage(callRes), nil
			},
		})
	}

	return nil
}

func (p *ProcessPlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, err := p.transport.Call("start", nil)
	return err
}

func (p *ProcessPlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	// Try graceful stop over RPC
	done := make(chan struct{})
	go func() {
		if p.transport != nil {
			_, _ = p.transport.Call("stop", nil)
			_ = p.transport.Close()
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		// Grace period expired, force kill
		_ = p.cmd.Process.Kill()
	}

	_ = p.stdin.Close()
	_ = p.stdout.Close()
	_ = p.cmd.Wait()
	return nil
}
