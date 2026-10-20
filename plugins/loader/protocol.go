package loader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// Standard JSON-RPC 2.0 error codes
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc error [%d]: %s", e.Code, e.Message)
}

// Transport handles line-delimited JSON-RPC message framing over stdin/stdout.
type Transport struct {
	reader  *bufio.Reader
	writer  io.Writer
	mu      sync.Mutex
	pending map[string]chan *RPCResponse
	seq     uint64
	done    chan struct{}
}

func NewTransport(r io.Reader, w io.Writer) *Transport {
	t := &Transport{
		reader:  bufio.NewReader(r),
		writer:  w,
		pending: make(map[string]chan *RPCResponse),
		done:    make(chan struct{}),
	}
	go t.readLoop()
	return t
}

func (t *Transport) NextID() string {
	val := atomic.AddUint64(&t.seq, 1)
	return fmt.Sprintf("%d", val)
}

func (t *Transport) readLoop() {
	for {
		line, err := t.reader.ReadBytes('\n')
		if err != nil {
			close(t.done)
			return
		}

		var resp RPCResponse
		if err := json.Unmarshal(line, &resp); err == nil && resp.ID != "" {
			t.mu.Lock()
			ch, ok := t.pending[resp.ID]
			if ok {
				delete(t.pending, resp.ID)
			}
			t.mu.Unlock()

			if ok {
				ch <- &resp
			}
		}
	}
}

// Call sends a request and waits for a response.
func (t *Transport) Call(method string, params any) (json.RawMessage, error) {
	id := t.NextID()

	var paramsBytes json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		paramsBytes = b
	}

	req := RPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsBytes,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	reqBytes = append(reqBytes, '\n')

	ch := make(chan *RPCResponse, 1)
	t.mu.Lock()
	t.pending[id] = ch
	t.mu.Unlock()

	t.mu.Lock()
	_, err = t.writer.Write(reqBytes)
	t.mu.Unlock()

	if err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("failed writing rpc request: %w", err)
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	case <-t.done:
		return nil, fmt.Errorf("plugin process transport closed")
	}
}

// Close closes the transport.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	for id, ch := range t.pending {
		delete(t.pending, id)
		close(ch)
	}
	return nil
}
