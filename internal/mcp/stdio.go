package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// ServeStdio runs the MCP server over stdin/stdout using newline-delimited JSON-RPC.
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

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if len(line) == 0 {
			continue
		}
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
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
