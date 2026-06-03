package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// readResult carries a single line (or error) from the background reader goroutine.
type readResult struct {
	line []byte
	err  error
}

// ServeStdio runs the MCP server over stdin/stdout using newline-delimited JSON-RPC.
// A background goroutine performs the blocking read so that context cancellation
// (e.g. from Ctrl+C / SIGINT) can interrupt the loop promptly.
func (s *Server) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	var writeMu sync.Mutex

	writeJSON := func(v any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := out.Write(append(b, '\n')); err != nil {
			return err
		}
		return nil
	}

	// Allow server-initiated notifications.
	s.SetNotifyFn(func(method string, params any) {
		_ = writeJSON(&Notification{JSONRPC: JSONRPCVersion, Method: method, Params: params})
	})

	// Run the blocking ReadBytes in a dedicated goroutine so we can
	// select on ctx.Done() without being stuck in a syscall.
	ch := make(chan readResult, 1)
	go func() {
		for {
			line, err := reader.ReadBytes('\n')
			ch <- readResult{line, err}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case res := <-ch:
			if res.err != nil {
				if res.err == io.EOF {
					return nil
				}
				return res.err
			}
			if len(res.line) == 0 {
				continue
			}
			var req Request
			if err := json.Unmarshal(res.line, &req); err != nil {
				_ = writeJSON(&Response{
					JSONRPC: JSONRPCVersion,
					Error:   &RPCError{Code: ErrCodeParse, Message: err.Error()},
				})
				continue
			}
			resp := s.HandleRequest(ctx, &req)
			if resp != nil {
				if err := writeJSON(resp); err != nil {
					return fmt.Errorf("failed to write response: %w", err)
				}
			}
		}
	}
}
