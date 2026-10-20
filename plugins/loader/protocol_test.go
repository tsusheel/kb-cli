package loader

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

// pipeReadWriter pairs two pipes for full duplex communication in memory.
type pipeReadWriter struct {
	io.Reader
	io.Writer
}

func TestTransport_Roundtrip(t *testing.T) {
	// Simulated plugin stdin/stdout pipes
	rHost, wPlugin := io.Pipe()
	rPlugin, wHost := io.Pipe()

	hostTransport := NewTransport(rHost, wHost)
	defer hostTransport.Close()

	// Simulated external plugin server loop
	go func() {
		dec := json.NewDecoder(rPlugin)
		for {
			var req RPCRequest
			if err := dec.Decode(&req); err != nil {
				return
			}

			if req.Method == "echo" {
				resp := RPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result:  req.Params,
				}
				data, _ := json.Marshal(resp)
				data = append(data, '\n')
				_, _ = wPlugin.Write(data)
			}
		}
	}()

	res, err := hostTransport.Call("echo", map[string]string{"message": "ping"})
	if err != nil {
		t.Fatalf("Transport.Call failed: %v", err)
	}

	var output map[string]string
	if err := json.Unmarshal(res, &output); err != nil {
		t.Fatalf("failed decoding response: %v", err)
	}

	if output["message"] != "ping" {
		t.Errorf("expected echo 'ping', got %v", output)
	}
}

func TestTransport_RPCError(t *testing.T) {
	rHost, wPlugin := io.Pipe()
	rPlugin, wHost := io.Pipe()

	hostTransport := NewTransport(rHost, wHost)
	defer hostTransport.Close()

	go func() {
		dec := json.NewDecoder(rPlugin)
		var req RPCRequest
		_ = dec.Decode(&req)

		resp := RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    CodeMethodNotFound,
				Message: "Method not found",
			},
		}
		data, _ := json.Marshal(resp)
		data = append(data, '\n')
		_, _ = wPlugin.Write(data)
	}()

	_, err := hostTransport.Call("nonexistent_method", nil)
	if err == nil {
		t.Fatalf("expected error for nonexistent method, got nil")
	}

	rpcErr, ok := err.(*RPCError)
	if !ok || rpcErr.Code != CodeMethodNotFound {
		t.Errorf("expected RPCError with code %d, got %v", CodeMethodNotFound, err)
	}
}

func TestRPCRequestSerialization(t *testing.T) {
	req := RPCRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "test",
		Params:  json.RawMessage(`{"key":"value"}`),
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(req); err != nil {
		t.Fatalf("failed encoding request: %v", err)
	}

	var decoded RPCRequest
	if err := json.NewDecoder(buf).Decode(&decoded); err != nil {
		t.Fatalf("failed decoding request: %v", err)
	}

	if decoded.Method != "test" || decoded.ID != "1" {
		t.Errorf("unexpected decoded request: %+v", decoded)
	}
}
