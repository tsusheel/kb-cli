// dummy_plugin is a minimal external plugin binary for end-to-end testing of kb-cli's
// plugin loader. It implements the JSON-RPC 2.0 stdio protocol expected by the host.
//
// Supported methods:
//   - handshake → returns manifest
//   - register  → returns CLI commands + MCP tools
//   - command.execute → echoes "Hello from dummy plugin!"
//   - mcp.call  → echoes the arguments back
//   - start     → no-op
//   - stop      → exits cleanly
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func respond(w *bufio.Writer, id string, result any) {
	data, _ := json.Marshal(result)
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  data,
	}
	b, _ := json.Marshal(resp)
	_, _ = w.Write(b)
	_, _ = w.Write([]byte("\n"))
	_ = w.Flush()
}

func respondError(w *bufio.Writer, id string, code int, msg string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	}
	b, _ := json.Marshal(resp)
	_, _ = w.Write(b)
	_, _ = w.Write([]byte("\n"))
	_ = w.Flush()
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	// Allow large messages (1MB)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	writer := bufio.NewWriter(os.Stdout)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			respondError(writer, "", -32700, fmt.Sprintf("parse error: %v", err))
			continue
		}

		switch req.Method {
		case "handshake":
			manifest := map[string]any{
				"id":               "dummy",
				"name":             "Dummy Test Plugin",
				"version":          "0.1.0",
				"api_version":      "1",
				"schema_version":   1,
				"min_core_version": "0.1.0",
				"capabilities":     []string{"storage.entries", "cli.commands", "mcp.tools"},
			}
			respond(writer, req.ID, manifest)

		case "register":
			result := map[string]any{
				"commands": []map[string]string{
					{
						"use":   "dummy-hello",
						"short": "Say hello from the dummy plugin",
						"long":  "A test command that prints a greeting from the dummy external plugin.",
					},
				},
				"mcp_tools": []map[string]any{
					{
						"name":        "dummy_echo",
						"description": "Echoes the input back",
						"schema_json": json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"}}}`),
					},
				},
			}
			respond(writer, req.ID, result)

		case "command.execute":
			var params struct {
				Command string   `json:"command"`
				Args    []string `json:"args"`
			}
			_ = json.Unmarshal(req.Params, &params)

			output := fmt.Sprintf("Hello from dummy plugin! command=%s args=%v\n", params.Command, params.Args)
			respond(writer, req.ID, output)

		case "mcp.call":
			var params struct {
				Tool string          `json:"tool"`
				Args json.RawMessage `json:"args"`
			}
			_ = json.Unmarshal(req.Params, &params)

			result := map[string]any{
				"tool":   params.Tool,
				"echoed": json.RawMessage(params.Args),
			}
			respond(writer, req.ID, result)

		case "start":
			respond(writer, req.ID, map[string]string{"status": "ok"})

		case "stop":
			respond(writer, req.ID, map[string]string{"status": "stopped"})
			os.Exit(0)

		default:
			respondError(writer, req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
		}
	}
}
