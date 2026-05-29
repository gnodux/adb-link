package api

import (
	"net/http"

	"github.com/gnodux/adb-link/internal/services"
)

// NewRouter constructs the HTTP router with all API routes registered.
func NewRouter(c *services.Container) http.Handler {
	return NewRouterWithMCP(c, nil)
}

// NewRouterWithMCP is like NewRouter but additionally mounts an MCP HTTP
// handler on /mcp and /mcp/ when mcpHandler is non-nil.
func NewRouterWithMCP(c *services.Container, mcpHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	h := NewHandlers(c)

	// Health
	mux.HandleFunc("GET /api/health", h.Health)

	// Datasources
	mux.HandleFunc("POST /api/datasources/list", h.ListDatasources)
	mux.HandleFunc("POST /api/datasources/detail", h.DatasourceDetail)
	mux.HandleFunc("POST /api/datasources/test", h.DatasourceTest)

	// Schema
	mux.HandleFunc("POST /api/databases/list", h.ListDatabases)
	mux.HandleFunc("POST /api/schema/get", h.GetSchema)
	mux.HandleFunc("POST /api/schema/table", h.GetTableInfo)
	mux.HandleFunc("POST /api/schema/view", h.GetViewInfo)

	// Query
	mux.HandleFunc("POST /api/query/execute", h.ExecuteQuery)
	mux.HandleFunc("POST /api/query/explain", h.ExplainQuery)

	// Async query
	mux.HandleFunc("POST /api/async/query/submit", h.SubmitAsyncQuery)
	mux.HandleFunc("POST /api/async/query/status", h.AsyncQueryStatus)
	mux.HandleFunc("POST /api/async/query/result", h.AsyncQueryResult)
	mux.HandleFunc("POST /api/async/query/cancel", h.CancelAsyncQuery)

	// Tools
	mux.HandleFunc("GET /api/tools", h.ListTools)
	mux.HandleFunc("POST /api/tool/register", h.RegisterTool)
	mux.HandleFunc("POST /api/tool/unregister", h.UnregisterTool)

	// Async tools (path params)
	mux.HandleFunc("POST /api/tool/async/{tool_name}/submit", h.SubmitAsyncTool)
	mux.HandleFunc("POST /api/tool/async/{tool_name}/status", h.AsyncToolStatus)
	mux.HandleFunc("POST /api/tool/async/{tool_name}/result", h.AsyncToolResult)
	mux.HandleFunc("POST /api/tool/async/{tool_name}/cancel", h.CancelAsyncTool)

	// Dynamic tool execution
	mux.HandleFunc("POST /api/tool/{tool_name}", h.ExecuteTool)

	// MCP HTTP transport
	if mcpHandler != nil {
		mux.Handle("/mcp", mcpHandler)
		mux.Handle("/mcp/", mcpHandler)
	}

	// Compose middleware: CORS -> auth -> mux
	var handler http.Handler = mux
	handler = BearerAuth(c.ConfigService)(handler)
	handler = CORS(handler)
	return handler
}
