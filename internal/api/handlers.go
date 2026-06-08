package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gnodux/adb-link/internal/models"
	"github.com/gnodux/adb-link/internal/services"
)

// Handlers holds dependencies needed by HTTP handlers.
type Handlers struct {
	Container *services.Container
}

// NewHandlers constructs Handlers around a service container.
func NewHandlers(c *services.Container) *Handlers {
	return &Handlers{Container: c}
}

// --- Datasources ---

type datasourceNameRequest struct {
	Name string `json:"name"`
}

func (h *Handlers) ListDatasources(w http.ResponseWriter, r *http.Request) {
	user := UserNameFromRequest(r)
	all := h.Container.ConfigService.ListDatasources()
	filtered := make([]models.DatasourceInfo, 0, len(all))
	for _, ds := range all {
		if h.Container.PermissionService.CheckDatasource(user, ds.Name) {
			// Enrich with server info (cached on first connection)
			if info, err := h.Container.ConnectionService.GetServerInfo(r.Context(), ds.Name); err == nil {
				ds.ServerInfo = info
			}
			filtered = append(filtered, ds)
		}
	}
	WriteOK(w, filtered)
}

func (h *Handlers) DatasourceDetail(w http.ResponseWriter, r *http.Request) {
	var req datasourceNameRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	cfg, err := h.Container.ConfigService.GetDatasource(req.Name)
	if err != nil {
		WriteErrorStatus(w, http.StatusNotFound, err.Error())
		return
	}
	// Mask password
	masked := *cfg
	conn := masked.Connection
	if conn.Password != "" {
		conn.Password = "***"
	}
	masked.Connection = conn
	WriteOK(w, masked)
}

func (h *Handlers) DatasourceTest(w http.ResponseWriter, r *http.Request) {
	var req datasourceNameRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	cfg, err := h.Container.ConfigService.GetDatasource(req.Name)
	if err != nil {
		WriteJSON(w, http.StatusOK, models.APIResponse{
			Success: false, Data: map[string]bool{"connected": false}, Error: err.Error(),
		})
		return
	}
	if services.IsNonSQLType(cfg.Type) {
		client, _, err := h.Container.ConnectionService.GetNonSQLClient(req.Name)
		if err != nil {
			WriteJSON(w, http.StatusOK, models.APIResponse{
				Success: false, Data: map[string]bool{"connected": false}, Error: err.Error(),
			})
			return
		}
		if err := client.Ping(r.Context()); err != nil {
			WriteJSON(w, http.StatusOK, models.APIResponse{
				Success: false, Data: map[string]bool{"connected": false}, Error: err.Error(),
			})
			return
		}
	} else {
		db, _, err := h.Container.ConnectionService.GetSQLDB(req.Name, "")
		if err != nil {
			WriteJSON(w, http.StatusOK, models.APIResponse{
				Success: false, Data: map[string]bool{"connected": false}, Error: err.Error(),
			})
			return
		}
		if err := db.PingContext(r.Context()); err != nil {
			WriteJSON(w, http.StatusOK, models.APIResponse{
				Success: false, Data: map[string]bool{"connected": false}, Error: err.Error(),
			})
			return
		}
	}
	WriteOK(w, map[string]bool{"connected": true})
}

// --- Schema ---

type listDatabasesRequest struct {
	DatasourceName string `json:"datasource_name"`
}

type getSchemaRequest struct {
	DatasourceName string `json:"datasource_name"`
	Database       string `json:"database"`
}

type getTableRequest struct {
	DatasourceName string `json:"datasource_name"`
	Database       string `json:"database"`
	Table          string `json:"table"`
}

type getViewRequest struct {
	DatasourceName string `json:"datasource_name"`
	Database       string `json:"database"`
	View           string `json:"view"`
}

func (h *Handlers) ListDatabases(w http.ResponseWriter, r *http.Request) {
	var req listDatabasesRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	user := UserNameFromRequest(r)
	dbs, err := h.Container.SchemaService.GetDatabases(r.Context(), req.DatasourceName, user)
	if err != nil {
		WriteAppError(w, err)
		return
	}
	WriteOK(w, dbs)
}

func (h *Handlers) GetSchema(w http.ResponseWriter, r *http.Request) {
	var req getSchemaRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	user := UserNameFromRequest(r)
	schema, err := h.Container.SchemaService.GetSchema(r.Context(), req.DatasourceName, req.Database, user)
	if err != nil {
		WriteAppError(w, err)
		return
	}
	WriteOK(w, schema)
}

func (h *Handlers) GetTableInfo(w http.ResponseWriter, r *http.Request) {
	var req getTableRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	user := UserNameFromRequest(r)
	info, err := h.Container.SchemaService.GetTableInfo(r.Context(), req.DatasourceName, req.Database, req.Table, user)
	if err != nil {
		WriteAppError(w, err)
		return
	}
	WriteOK(w, info)
}

func (h *Handlers) GetViewInfo(w http.ResponseWriter, r *http.Request) {
	var req getViewRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	user := UserNameFromRequest(r)
	info, err := h.Container.SchemaService.GetViewInfo(r.Context(), req.DatasourceName, req.Database, req.View, user)
	if err != nil {
		WriteAppError(w, err)
		return
	}
	WriteOK(w, info)
}

// --- Query ---

type executeQueryRequest struct {
	DatasourceName string `json:"datasource_name"`
	Database       string `json:"database"`
	SQL            string `json:"sql"`
	Limit          int    `json:"limit"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type explainQueryRequest struct {
	DatasourceName string `json:"datasource_name"`
	Database       string `json:"database"`
	SQL            string `json:"sql"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

func (h *Handlers) ExecuteQuery(w http.ResponseWriter, r *http.Request) {
	var req executeQueryRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	if req.Limit <= 0 {
		req.Limit = 100
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 30
	}
	user := UserNameFromRequest(r)
	result, err := h.Container.QueryService.Execute(r.Context(), &models.QueryRequest{
		DatasourceName: req.DatasourceName,
		Database:       req.Database,
		SQL:            req.SQL,
		Limit:          req.Limit,
		TimeoutSeconds: req.TimeoutSeconds,
	}, user)
	if err != nil {
		WriteAppError(w, err)
		return
	}
	WriteOK(w, result)
}

func (h *Handlers) ExplainQuery(w http.ResponseWriter, r *http.Request) {
	var req explainQueryRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 30
	}
	user := UserNameFromRequest(r)
	result, err := h.Container.QueryService.Explain(r.Context(), &models.ExplainRequest{
		DatasourceName: req.DatasourceName,
		Database:       req.Database,
		SQL:            req.SQL,
		TimeoutSeconds: req.TimeoutSeconds,
	}, user)
	if err != nil {
		if _, ok := err.(*services.UnsupportedOperationError); ok {
			WriteErrorStatus(w, http.StatusBadRequest, "unsupported: "+err.Error())
			return
		}
		WriteAppError(w, err)
		return
	}
	WriteOK(w, result)
}

// --- Async Query ---

func (h *Handlers) SubmitAsyncQuery(w http.ResponseWriter, r *http.Request) {
	var req models.AsyncQueryRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	user := UserNameFromRequest(r)
	cfg, err := h.Container.ConfigService.GetDatasource(req.DatasourceName)
	if err != nil {
		WriteErrorStatus(w, http.StatusNotFound, err.Error())
		return
	}
	if cfg.Shadow {
		WriteErrorStatus(w, http.StatusForbidden,
			"数据源 '"+req.DatasourceName+"' 是 shadow 数据源，不允许直接查询。请通过已配置的工具(tool)来访问该数据源。")
		return
	}
	queryID, err := h.Container.AsyncQueryService.Submit(&req, user)
	if err != nil {
		WriteAppError(w, err)
		return
	}
	WriteOK(w, map[string]string{"query_id": queryID})
}

type queryIDRequest struct {
	QueryID string `json:"query_id"`
}

func (h *Handlers) AsyncQueryStatus(w http.ResponseWriter, r *http.Request) {
	var req queryIDRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	status, err := h.Container.AsyncQueryService.GetStatus(req.QueryID)
	if err != nil {
		WriteErrorStatus(w, http.StatusNotFound, err.Error())
		return
	}
	WriteOK(w, status)
}

func (h *Handlers) AsyncQueryResult(w http.ResponseWriter, r *http.Request) {
	var req queryIDRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	result, err := h.Container.AsyncQueryService.GetResult(req.QueryID)
	if err != nil {
		WriteErrorStatus(w, http.StatusNotFound, err.Error())
		return
	}
	WriteOK(w, result)
}

func (h *Handlers) CancelAsyncQuery(w http.ResponseWriter, r *http.Request) {
	var req queryIDRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	if err := h.Container.AsyncQueryService.Cancel(req.QueryID); err != nil {
		WriteErrorStatus(w, http.StatusNotFound, err.Error())
		return
	}
	WriteOK(w, map[string]string{"query_id": req.QueryID})
}

// --- Async Tool ---

func (h *Handlers) extractToolName(r *http.Request, prefix string) string {
	// path is /api/tool/async/{tool_name}/submit
	path := strings.TrimPrefix(r.URL.Path, prefix)
	idx := strings.Index(path, "/")
	if idx < 0 {
		return path
	}
	return path[:idx]
}

func (h *Handlers) SubmitAsyncTool(w http.ResponseWriter, r *http.Request) {
	toolName := r.PathValue("tool_name")
	user := UserNameFromRequest(r)
	if !h.Container.PermissionService.CheckTool(user, toolName) {
		WriteErrorStatus(w, http.StatusForbidden,
			"Access denied: user '"+user+"' cannot execute tool '"+toolName+"'")
		return
	}
	var req models.AsyncToolRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	queryID, err := h.Container.AsyncQueryService.SubmitTool(toolName, req.Parameters, req.TimeoutSeconds, user)
	if err != nil {
		WriteAppError(w, err)
		return
	}
	WriteOK(w, map[string]string{"query_id": queryID})
}

func (h *Handlers) AsyncToolStatus(w http.ResponseWriter, r *http.Request) {
	h.AsyncQueryStatus(w, r)
}

func (h *Handlers) AsyncToolResult(w http.ResponseWriter, r *http.Request) {
	h.AsyncQueryResult(w, r)
}

func (h *Handlers) CancelAsyncTool(w http.ResponseWriter, r *http.Request) {
	h.CancelAsyncQuery(w, r)
}

// --- Tools ---

type toolListItem struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Datasource  string                 `json:"datasource"`
	Database    string                 `json:"database,omitempty"`
	Parameters  []models.ToolParameter `json:"parameters"`
}

type toolsetListItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
}

func (h *Handlers) ListTools(w http.ResponseWriter, r *http.Request) {
	user := UserNameFromRequest(r)
	tools := []toolListItem{}
	for _, tc := range h.Container.ConfigService.AllTools() {
		if !h.Container.PermissionService.CheckTool(user, tc.Name) {
			continue
		}
		tools = append(tools, toolListItem{
			Name:        tc.Name,
			Description: tc.Description,
			Datasource:  tc.Datasource,
			Database:    tc.Database,
			Parameters:  tc.Parameters,
		})
	}
	toolsets := []toolsetListItem{}
	for _, ts := range h.Container.ConfigService.AllToolsets() {
		toolsets = append(toolsets, toolsetListItem{
			Name:        ts.Name,
			Description: ts.Description,
			Tools:       ts.Tools,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"tools":    tools,
			"toolsets": toolsets,
		},
	})
}

type toolExecuteRequest struct {
	Parameters map[string]any `json:"parameters"`
}

func (h *Handlers) ExecuteTool(w http.ResponseWriter, r *http.Request) {
	toolName := r.PathValue("tool_name")
	user := UserNameFromRequest(r)
	tool, err := h.Container.ConfigService.GetTool(toolName)
	if err != nil {
		WriteErrorStatus(w, http.StatusNotFound, err.Error())
		return
	}
	if !h.Container.PermissionService.CheckTool(user, toolName) {
		WriteErrorStatus(w, http.StatusForbidden,
			"Access denied: user '"+user+"' cannot execute tool '"+toolName+"'")
		return
	}
	var req toolExecuteRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	result, err := h.Container.QueryService.ExecuteTemplate(r.Context(), tool, req.Parameters, user)
	if err != nil {
		WriteErrorStatus(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": result})
}

type toolRegisterRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Datasource  string          `json:"datasource"`
	Database    string          `json:"database,omitempty"`
	Template    string          `json:"template"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type toolUnregisterRequest struct {
	Name string `json:"name"`
}

func (h *Handlers) RegisterTool(w http.ResponseWriter, r *http.Request) {
	var req toolRegisterRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	if _, err := h.Container.ConfigService.GetTool(req.Name); err == nil {
		WriteErrorStatus(w, http.StatusConflict,
			"Tool '"+req.Name+"' already exists. Use /tool/unregister first.")
		return
	}
	if _, err := h.Container.ConfigService.GetDatasource(req.Datasource); err != nil {
		WriteErrorStatus(w, http.StatusNotFound,
			"Datasource '"+req.Datasource+"' not found")
		return
	}
	// Validate input_schema is a valid JSON object
	if len(req.InputSchema) > 0 {
		var schemaMap map[string]any
		if err := json.Unmarshal(req.InputSchema, &schemaMap); err != nil {
			WriteErrorStatus(w, http.StatusBadRequest,
				"Invalid input_schema: "+err.Error())
			return
		}
	}
	tool := &models.ToolConfig{
		Kind:        "tool",
		Name:        req.Name,
		Description: req.Description,
		Datasource:  req.Datasource,
		Database:    req.Database,
		Template:    req.Template,
		InputSchema: req.InputSchema,
	}
	h.Container.ConfigService.RegisterTool(tool)
	filePath, err := h.Container.ConfigService.PersistTool(tool)
	if err != nil {
		WriteErrorStatus(w, http.StatusInternalServerError, err.Error())
		return
	}
	user := UserNameFromRequest(r)
	services.AuditLog().Printf("user=%s | action=register_tool | tool=%s | datasource=%s",
		user, req.Name, req.Datasource)
	WriteOK(w, map[string]string{"name": req.Name, "persisted_to": filePath})
}

func (h *Handlers) UnregisterTool(w http.ResponseWriter, r *http.Request) {
	var req toolUnregisterRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	if _, err := h.Container.ConfigService.GetTool(req.Name); err != nil {
		WriteErrorStatus(w, http.StatusNotFound, "Tool '"+req.Name+"' not found")
		return
	}
	h.Container.ConfigService.UnregisterTool(req.Name)
	h.Container.ConfigService.RemoveToolFile(req.Name)
	user := UserNameFromRequest(r)
	services.AuditLog().Printf("user=%s | action=unregister_tool | tool=%s", user, req.Name)
	WriteOK(w, map[string]string{"name": req.Name})
}

// --- Dynamic Datasource Registration ---

type datasourceRegisterRequest struct {
	Name        string                  `json:"name"`
	Type        models.DatabaseType     `json:"type"`
	Description string                  `json:"description,omitempty"`
	Connection  models.ConnectionConfig `json:"connection"`
	Options     map[string]any          `json:"options,omitempty"`
}

type datasourceUnregisterRequest struct {
	Name string `json:"name"`
}

func (h *Handlers) RegisterDatasource(w http.ResponseWriter, r *http.Request) {
	var req datasourceRegisterRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	if req.Name == "" {
		WriteErrorStatus(w, http.StatusBadRequest, "name is required")
		return
	}
	if _, err := h.Container.ConfigService.GetDatasource(req.Name); err == nil {
		WriteErrorStatus(w, http.StatusConflict,
			"Datasource '"+req.Name+"' already exists. Use /datasources/unregister first.")
		return
	}
	cfg := &models.DatasourceConfig{
		Kind:        "datasource",
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Connection:  req.Connection,
		Options:     req.Options,
	}
	// Register to snapshot first so ConnectionService can resolve it.
	h.Container.ConfigService.RegisterDatasource(cfg)

	// Validate connection.
	if services.IsNonSQLType(cfg.Type) {
		client, _, err := h.Container.ConnectionService.GetNonSQLClient(req.Name)
		if err != nil {
			h.Container.ConfigService.UnregisterDatasource(req.Name)
			WriteErrorStatus(w, http.StatusBadRequest,
				"Connection validation failed: "+err.Error())
			return
		}
		if err := client.Ping(r.Context()); err != nil {
			h.Container.ConfigService.UnregisterDatasource(req.Name)
			h.Container.ConnectionService.Invalidate(req.Name)
			WriteErrorStatus(w, http.StatusBadRequest,
				"Connection validation failed: "+err.Error())
			return
		}
	} else {
		db, _, err := h.Container.ConnectionService.GetSQLDB(req.Name, "")
		if err != nil {
			h.Container.ConfigService.UnregisterDatasource(req.Name)
			WriteErrorStatus(w, http.StatusBadRequest,
				"Connection validation failed: "+err.Error())
			return
		}
		if err := db.PingContext(r.Context()); err != nil {
			h.Container.ConfigService.UnregisterDatasource(req.Name)
			h.Container.ConnectionService.Invalidate(req.Name)
			WriteErrorStatus(w, http.StatusBadRequest,
				"Connection validation failed: "+err.Error())
			return
		}
	}
	// Release the validation connection; it will be re-opened on first use.
	h.Container.ConnectionService.Invalidate(req.Name)

	filePath, err := h.Container.ConfigService.PersistDatasource(cfg)
	if err != nil {
		WriteErrorStatus(w, http.StatusInternalServerError, err.Error())
		return
	}
	user := UserNameFromRequest(r)
	services.AuditLog().Printf("user=%s | action=register_datasource | datasource=%s | type=%s",
		user, req.Name, req.Type)
	WriteOK(w, map[string]string{"name": req.Name, "persisted_to": filePath})
}

func (h *Handlers) UnregisterDatasource(w http.ResponseWriter, r *http.Request) {
	var req datasourceUnregisterRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, err.Error())
		return
	}
	if _, err := h.Container.ConfigService.GetDatasource(req.Name); err != nil {
		WriteErrorStatus(w, http.StatusNotFound, "Datasource '"+req.Name+"' not found")
		return
	}
	h.Container.ConfigService.UnregisterDatasource(req.Name)
	h.Container.ConnectionService.Invalidate(req.Name)
	h.Container.ConfigService.RemoveDatasourceFile(req.Name)
	user := UserNameFromRequest(r)
	services.AuditLog().Printf("user=%s | action=unregister_datasource | datasource=%s", user, req.Name)
	WriteOK(w, map[string]string{"name": req.Name})
}

// Health is the liveness endpoint.
func (h *Handlers) Health(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
