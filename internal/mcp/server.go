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
	ErrCodeParse             = -32700
	ErrCodeInvalidRequest    = -32600
	ErrCodeMethodNotFound    = -32601
	ErrCodeInvalidParams     = -32602
	ErrCodeInternal          = -32603
	ErrCodeResourceNotFound  = -32002
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

// Resource describes an MCP resource exposed to clients.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceTemplate describes a parameterized MCP resource.
type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceHandler resolves a URI into resource contents.
type ResourceHandler func(ctx context.Context, uri string) ([]ResourceContent, error)

// ResourceContent is one content block in a resources/read response.
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

// resourceEntry pairs a resource definition with its handler.
type resourceEntry struct {
	resource Resource
	handler  ResourceHandler
}

// templateEntry pairs a resource template definition with its handler.
type templateEntry struct {
	template ResourceTemplate
	handler  ResourceHandler
}

// Server is an MCP server with tool and resource registries.
type Server struct {
	mu        sync.RWMutex
	name      string
	version   string
	tools     map[string]*toolEntry
	resources map[string]*resourceEntry
	templates []templateEntry
	notifyFn  func(method string, params any)
}

// NewServer creates a new MCP server.
func NewServer(name, version string) *Server {
	return &Server{
		name:      name,
		version:   version,
		tools:     make(map[string]*toolEntry),
		resources: make(map[string]*resourceEntry),
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

// RegisterResource registers a static resource with its handler.
func (s *Server) RegisterResource(r Resource, handler ResourceHandler) {
	s.mu.Lock()
	s.resources[r.URI] = &resourceEntry{resource: r, handler: handler}
	s.mu.Unlock()
	s.notifyResourceListChanged()
}

// RegisterResourceTemplate registers a URI template with its handler.
func (s *Server) RegisterResourceTemplate(t ResourceTemplate, handler ResourceHandler) {
	s.mu.Lock()
	s.templates = append(s.templates, templateEntry{template: t, handler: handler})
	s.mu.Unlock()
	s.notifyResourceListChanged()
}

// NotifyResourceListChanged sends a resources/list_changed notification.
// Exported so the container can call it on config reload.
func (s *Server) NotifyResourceListChanged() {
	s.notifyResourceListChanged()
}

func (s *Server) notifyResourceListChanged() {
	s.mu.RLock()
	fn := s.notifyFn
	s.mu.RUnlock()
	if fn != nil {
		fn("notifications/resources/list_changed", nil)
	}
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
	case "resources/list":
		return s.handleResourcesList(req)
	case "resources/templates/list":
		return s.handleResourcesTemplatesList(req)
	case "resources/read":
		return s.handleResourcesRead(ctx, req)
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
			"resources": map[string]any{
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

func (s *Server) handleResourcesList(req *Request) *Response {
	s.mu.RLock()
	resources := make([]Resource, 0, len(s.resources))
	for _, e := range s.resources {
		resources = append(resources, e.resource)
	}
	s.mu.RUnlock()
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  map[string]any{"resources": resources},
	}
}

func (s *Server) handleResourcesTemplatesList(req *Request) *Response {
	s.mu.RLock()
	templates := make([]ResourceTemplate, 0, len(s.templates))
	for _, e := range s.templates {
		templates = append(templates, e.template)
	}
	s.mu.RUnlock()
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  map[string]any{"resourceTemplates": templates},
	}
}

type resourcesReadParams struct {
	URI string `json:"uri"`
}

func (s *Server) handleResourcesRead(ctx context.Context, req *Request) *Response {
	var params resourcesReadParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   &RPCError{Code: ErrCodeInvalidParams, Message: err.Error()},
		}
	}

	s.mu.RLock()
	// Check static resources first.
	if entry, ok := s.resources[params.URI]; ok {
		handler := entry.handler
		s.mu.RUnlock()
		contents, err := handler(ctx, params.URI)
		if err != nil {
			return &Response{
				JSONRPC: JSONRPCVersion,
				ID:      req.ID,
				Error:   &RPCError{Code: ErrCodeInternal, Message: err.Error()},
			}
		}
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Result:  map[string]any{"contents": contents},
		}
	}

	// Try matching against templates.
	for _, te := range s.templates {
		if matched, _ := matchTemplate(params.URI, te.template.URITemplate); matched {
			handler := te.handler
			s.mu.RUnlock()
			contents, err := handler(ctx, params.URI)
			if err != nil {
				return &Response{
					JSONRPC: JSONRPCVersion,
					ID:      req.ID,
					Error:   &RPCError{Code: ErrCodeInternal, Message: err.Error()},
				}
			}
			return &Response{
				JSONRPC: JSONRPCVersion,
				ID:      req.ID,
				Result:  map[string]any{"contents": contents},
			}
		}
	}
	s.mu.RUnlock()

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Error:   &RPCError{Code: ErrCodeResourceNotFound, Message: "resource not found: " + params.URI},
	}
}

// matchTemplate checks if a URI matches a template pattern and extracts segments.
// Template segments wrapped in braces (e.g. {name}) are wildcards.
func matchTemplate(uri, tmpl string) (bool, []string) {
	// Strip scheme: both should share the same "datasource:///" prefix style.
	uriPath := stripScheme(uri)
	tmplPath := stripScheme(tmpl)

	uriParts := splitPath(uriPath)
	tmplParts := splitPath(tmplPath)

	if len(uriParts) != len(tmplParts) {
		return false, nil
	}
	var captured []string
	for i, tp := range tmplParts {
		if len(tp) > 2 && tp[0] == '{' && tp[len(tp)-1] == '}' {
			captured = append(captured, uriParts[i])
		} else if tp != uriParts[i] {
			return false, nil
		}
	}
	return true, captured
}

func stripScheme(uri string) string {
	if idx := len("datasource:///"); len(uri) >= idx && uri[:idx] == "datasource:///" {
		return uri[idx:]
	}
	// Fallback: find "://" and skip past it + trailing slash.
	for i := 0; i+3 <= len(uri); i++ {
		if uri[i] == ':' && uri[i+1] == '/' && uri[i+2] == '/' {
			rest := uri[i+3:]
			for len(rest) > 0 && rest[0] == '/' {
				rest = rest[1:]
			}
			return rest
		}
	}
	return uri
}

func splitPath(p string) []string {
	var parts []string
	for _, s := range split(p, '/') {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return parts
}

func split(s string, sep byte) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}
