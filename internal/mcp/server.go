// Package mcp implements a minimal Model Context Protocol (MCP) server
// providing JSON-RPC 2.0 over stdio and streamable HTTP transports.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Protocol constants.
const (
	ProtocolVersion = "2024-11-05"
	JSONRPCVersion  = "2.0"
)

// Request is a JSON-RPC 2.0 request envelope.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response envelope.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// Notification is a JSON-RPC 2.0 notification (no ID).
type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	ErrCodeParse          = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)

// Tool describes an MCP tool exposed to clients.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// ToolHandler is the function signature for tool execution.
type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

// toolEntry pairs a tool definition with its handler.
type toolEntry struct {
	tool    Tool
	handler ToolHandler
}

// Server is an MCP server with a tool registry.
type Server struct {
	mu       sync.RWMutex
	name     string
	version  string
	tools    map[string]*toolEntry
	notifyFn func(method string, params any)
}

// NewServer creates a new MCP server.
func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		tools:   make(map[string]*toolEntry),
	}
}

// RegisterTool registers (or replaces) a tool.
func (s *Server) RegisterTool(tool Tool, handler ToolHandler) {
	s.mu.Lock()
	s.tools[tool.Name] = &toolEntry{tool: tool, handler: handler}
	s.mu.Unlock()
	s.notifyToolListChanged()
}

// UnregisterTool removes a tool.
func (s *Server) UnregisterTool(name string) bool {
	s.mu.Lock()
	_, ok := s.tools[name]
	if ok {
		delete(s.tools, name)
	}
	s.mu.Unlock()
	if ok {
		s.notifyToolListChanged()
	}
	return ok
}

// SetNotifyFn registers a notification callback to send messages back to clients.
func (s *Server) SetNotifyFn(fn func(method string, params any)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifyFn = fn
}

func (s *Server) notifyToolListChanged() {
	s.mu.RLock()
	fn := s.notifyFn
	s.mu.RUnlock()
	if fn != nil {
		fn("notifications/tools/list_changed", nil)
	}
}

// NotifyToolListChanged sends a tools/list_changed notification.
// Exported so the container can call it on config reload.
func (s *Server) NotifyToolListChanged() {
	s.notifyToolListChanged()
}

// HandleRequest dispatches a single JSON-RPC request.
// Returns nil response for notifications.
func (s *Server) HandleRequest(ctx context.Context, req *Request) *Response {
	if req.JSONRPC != JSONRPCVersion {
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   &RPCError{Code: ErrCodeInvalidRequest, Message: "invalid jsonrpc version"},
		}
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized", "notifications/initialized":
		return nil // notification; no response
	case "ping":
		return &Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		// Notifications have no ID and no response.
		if len(req.ID) == 0 {
			return nil
		}
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   &RPCError{Code: ErrCodeMethodNotFound, Message: "method not found: " + req.Method},
		}
	}
}

func (s *Server) handleInitialize(req *Request) *Response {
	result := map[string]any{
		"protocolVersion": ProtocolVersion,
		"serverInfo": map[string]any{
			"name":    s.name,
			"version": s.version,
		},
		"capabilities": map[string]any{
			"tools": map[string]any{
				"listChanged": true,
			},
		},
	}
	return &Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: result}
}

func (s *Server) handleToolsList(req *Request) *Response {
	s.mu.RLock()
	tools := make([]Tool, 0, len(s.tools))
	for _, e := range s.tools {
		tools = append(tools, e.tool)
	}
	s.mu.RUnlock()
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  map[string]any{"tools": tools},
	}
}

type toolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func (s *Server) handleToolsCall(ctx context.Context, req *Request) *Response {
	var params toolsCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   &RPCError{Code: ErrCodeInvalidParams, Message: err.Error()},
		}
	}
	s.mu.RLock()
	entry, ok := s.tools[params.Name]
	s.mu.RUnlock()
	if !ok {
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   &RPCError{Code: ErrCodeMethodNotFound, Message: "tool not found: " + params.Name},
		}
	}

	output, err := entry.handler(ctx, params.Arguments)
	if err != nil {
		// MCP convention: return tool error inside content with isError=true
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("Error: %s", err.Error())},
				},
				"isError": true,
			},
		}
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result: map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": output},
			},
		},
	}
}
