package integration_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gnodux/adb-link/internal/mcp"
	"github.com/gnodux/adb-link/internal/models"
	"github.com/gnodux/adb-link/internal/services"
)

// newMCPContainer builds a full services.Container wired to a real SQLite DB
// for end-to-end MCP resource testing.
// PermissionService is created with empty configs so that bypass logic works
// for resource handlers. QueryService uses nil to bypass its own permission checks.
func newMCPContainer(t *testing.T) (*services.Container, func()) {
	t.Helper()

	_, cs, _ := setupSQLite(t)
	conn := services.NewConnectionService(cs)
	// Create PermissionService with empty configs — resource handlers need
	// this to not be nil, and empty userName bypasses checks.
	permSvc := services.NewPermissionService(nil, nil)
	metaSvc := services.NewMetadataService(cs.AllMetadata())
	schemaSvc := services.NewSchemaService(cs, conn, metaSvc, permSvc)
	// QueryService gets nil to bypass its permission checks entirely
	// (matching the pattern in sqlite_test.go's newSQLiteServices).
	querySvc := services.NewQueryService(conn, cs, nil)
	asyncSvc := services.NewAsyncQueryService(querySvc, cs, 300)

	c := &services.Container{
		ConfigService:     cs,
		ConnectionService: conn,
		MetadataService:   metaSvc,
		PermissionService: permSvc,
		SchemaService:     schemaSvc,
		QueryService:      querySvc,
		AsyncQueryService: asyncSvc,
	}

	cleanup := func() {
		asyncSvc.Stop()
		_ = conn.DisposeAll()
	}
	return c, cleanup
}

// mcpRequest sends a JSON-RPC request to the MCP server and returns the response.
// Uses an empty user name so that PermissionService bypasses all checks
// (matching the "no authentication configured" production behavior).
func mcpRequest(t *testing.T, srv *mcp.Server, method string, id int, params any) *mcp.Response {
	t.Helper()
	idJSON, _ := json.Marshal(id)
	var paramsJSON json.RawMessage
	if params != nil {
		paramsJSON, _ = json.Marshal(params)
	}
	req := &mcp.Request{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  method,
		Params:  paramsJSON,
	}
	ctx := context.Background()
	// Empty user name = bypass permission checks (isBypassUser returns true).
	ctx = models.WithAuthUser(ctx, &models.AuthUser{Name: ""})
	return srv.HandleRequest(ctx, req)
}

// --- Integration Tests ---

func TestMCPResources_Initialize_AdvertisesResources(t *testing.T) {
	c, cleanup := newMCPContainer(t)
	defer cleanup()

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreResources(srv, c)

	resp := mcpRequest(t, srv, "initialize", 1, nil)
	if resp.Error != nil {
		t.Fatalf("initialize error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	caps := result["capabilities"].(map[string]any)
	resCap, ok := caps["resources"].(map[string]any)
	if !ok {
		t.Fatal("expected resources capability in initialize response")
	}
	if resCap["listChanged"] != true {
		t.Fatal("expected resources.listChanged=true")
	}
}

func TestMCPResources_List_ReturnsStaticResource(t *testing.T) {
	c, cleanup := newMCPContainer(t)
	defer cleanup()

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreResources(srv, c)

	resp := mcpRequest(t, srv, "resources/list", 1, nil)
	if resp.Error != nil {
		t.Fatalf("resources/list error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	resources := result["resources"].([]mcp.Resource)
	if len(resources) != 1 {
		t.Fatalf("expected 1 static resource, got %d", len(resources))
	}
	if resources[0].URI != "datasource:///" {
		t.Errorf("expected URI %q, got %q", "datasource:///", resources[0].URI)
	}
	if resources[0].Name != "Datasources" {
		t.Errorf("expected name %q, got %q", "Datasources", resources[0].Name)
	}
}

func TestMCPResources_TemplatesList_ReturnsAllTemplates(t *testing.T) {
	c, cleanup := newMCPContainer(t)
	defer cleanup()

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreResources(srv, c)

	resp := mcpRequest(t, srv, "resources/templates/list", 1, nil)
	if resp.Error != nil {
		t.Fatalf("resources/templates/list error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	templates := result["resourceTemplates"].([]mcp.ResourceTemplate)
	if len(templates) != 5 {
		t.Fatalf("expected 5 templates, got %d", len(templates))
	}

	// Verify all expected templates are present.
	expected := map[string]bool{
		"datasource:///{name}":                    false,
		"datasource:///{name}/{db}/databases":     false,
		"datasource:///{name}/{db}/schema":        false,
		"datasource:///{name}/{db}/tables/{table}": false,
		"datasource:///{name}/{db}/views/{view}":   false,
	}
	for _, tmpl := range templates {
		if _, ok := expected[tmpl.URITemplate]; ok {
			expected[tmpl.URITemplate] = true
		}
	}
	for uri, found := range expected {
		if !found {
			t.Errorf("template %q not found", uri)
		}
	}
}

func TestMCPResources_Read_DatasourceList(t *testing.T) {
	c, cleanup := newMCPContainer(t)
	defer cleanup()

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreResources(srv, c)

	resp := mcpRequest(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///",
	})
	if resp.Error != nil {
		t.Fatalf("resources/read error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	contents := result["contents"].([]mcp.ResourceContent)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}

	var datasources []models.DatasourceInfo
	if err := json.Unmarshal([]byte(contents[0].Text), &datasources); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(datasources) != 1 {
		t.Fatalf("expected 1 datasource, got %d", len(datasources))
	}
	if datasources[0].Name != "test-sqlite" {
		t.Errorf("expected datasource name %q, got %q", "test-sqlite", datasources[0].Name)
	}
	if string(datasources[0].Type) != "sqlite" {
		t.Errorf("expected type %q, got %q", "sqlite", datasources[0].Type)
	}
}

func TestMCPResources_Read_DatasourceDetail(t *testing.T) {
	c, cleanup := newMCPContainer(t)
	defer cleanup()

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreResources(srv, c)

	resp := mcpRequest(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-sqlite",
	})
	if resp.Error != nil {
		t.Fatalf("resources/read error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	contents := result["contents"].([]mcp.ResourceContent)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}

	var cfg models.DatasourceConfig
	if err := json.Unmarshal([]byte(contents[0].Text), &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if cfg.Name != "test-sqlite" {
		t.Errorf("expected name %q, got %q", "test-sqlite", cfg.Name)
	}
	if string(cfg.Type) != "sqlite" {
		t.Errorf("expected type %q, got %q", "sqlite", cfg.Type)
	}
	if cfg.Connection.Path == "" {
		t.Error("expected non-empty connection path")
	}
}

func TestMCPResources_Read_Databases(t *testing.T) {
	c, cleanup := newMCPContainer(t)
	defer cleanup()

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreResources(srv, c)

	resp := mcpRequest(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-sqlite/main/databases",
	})
	if resp.Error != nil {
		t.Fatalf("resources/read error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	contents := result["contents"].([]mcp.ResourceContent)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}

	var dbs []models.ObjectName
	if err := json.Unmarshal([]byte(contents[0].Text), &dbs); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(dbs) != 1 {
		t.Fatalf("expected 1 database, got %d", len(dbs))
	}
	if dbs[0].Name != "main" {
		t.Errorf("expected database name %q, got %q", "main", dbs[0].Name)
	}
}

func TestMCPResources_Read_Schema(t *testing.T) {
	c, cleanup := newMCPContainer(t)
	defer cleanup()

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreResources(srv, c)

	resp := mcpRequest(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-sqlite/main/schema",
	})
	if resp.Error != nil {
		t.Fatalf("resources/read error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	contents := result["contents"].([]mcp.ResourceContent)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}

	var schema models.DatabaseSchema
	if err := json.Unmarshal([]byte(contents[0].Text), &schema); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(schema.Tables) < 2 {
		t.Errorf("expected at least 2 tables, got %d", len(schema.Tables))
	}
	if len(schema.Views) < 1 {
		t.Errorf("expected at least 1 view, got %d", len(schema.Views))
	}

	// Verify table names.
	tableNames := map[string]bool{}
	for _, tbl := range schema.Tables {
		tableNames[tbl.Name] = true
	}
	if !tableNames["users"] || !tableNames["orders"] {
		t.Errorf("expected tables users and orders, got %v", tableNames)
	}

	// Verify view name.
	viewNames := map[string]bool{}
	for _, v := range schema.Views {
		viewNames[v.Name] = true
	}
	if !viewNames["user_order_summary"] {
		t.Errorf("expected view user_order_summary, got %v", viewNames)
	}
}

func TestMCPResources_Read_TableInfo(t *testing.T) {
	c, cleanup := newMCPContainer(t)
	defer cleanup()

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreResources(srv, c)

	resp := mcpRequest(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-sqlite/main/tables/users",
	})
	if resp.Error != nil {
		t.Fatalf("resources/read error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	contents := result["contents"].([]mcp.ResourceContent)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}

	var ti models.TableInfo
	if err := json.Unmarshal([]byte(contents[0].Text), &ti); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if ti.Name != "users" {
		t.Errorf("expected table name %q, got %q", "users", ti.Name)
	}
	if len(ti.Columns) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(ti.Columns))
	}

	// Verify column names.
	colNames := map[string]bool{}
	for _, col := range ti.Columns {
		colNames[col.Name] = true
	}
	for _, expected := range []string{"id", "name", "email", "created_at"} {
		if !colNames[expected] {
			t.Errorf("expected column %q not found", expected)
		}
	}
}

func TestMCPResources_Read_ViewInfo(t *testing.T) {
	c, cleanup := newMCPContainer(t)
	defer cleanup()

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreResources(srv, c)

	resp := mcpRequest(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-sqlite/main/views/user_order_summary",
	})
	if resp.Error != nil {
		t.Fatalf("resources/read error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	contents := result["contents"].([]mcp.ResourceContent)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}

	var vi models.TableInfo
	if err := json.Unmarshal([]byte(contents[0].Text), &vi); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if vi.Name != "user_order_summary" {
		t.Errorf("expected view name %q, got %q", "user_order_summary", vi.Name)
	}
	if len(vi.Columns) != 3 {
		t.Errorf("expected 3 view columns, got %d", len(vi.Columns))
	}
}

func TestMCPResources_Read_NotFound(t *testing.T) {
	c, cleanup := newMCPContainer(t)
	defer cleanup()

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreResources(srv, c)

	resp := mcpRequest(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///nonexistent-ds",
	})
	if resp.Error == nil {
		t.Fatal("expected error for nonexistent datasource")
	}
	// Should get an internal error (from service layer), not resource-not-found
	// because the template matched but the handler returned an error.
	if resp.Error.Code == 0 {
		t.Errorf("expected non-zero error code, got %d", resp.Error.Code)
	}
}

func TestMCPResources_Read_InvalidURI(t *testing.T) {
	c, cleanup := newMCPContainer(t)
	defer cleanup()

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreResources(srv, c)

	// URI that doesn't match any static resource or template.
	resp := mcpRequest(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///a/b/unknown-segment",
	})
	if resp.Error == nil {
		t.Fatal("expected error for unmatched URI")
	}
}

func TestMCPResources_ToolsStillWork(t *testing.T) {
	c, cleanup := newMCPContainer(t)
	defer cleanup()

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreTools(srv, c)
	mcp.RegisterCoreResources(srv, c)

	// Verify tools/list no longer contains the migrated read-only tools.
	resp := mcpRequest(t, srv, "tools/list", 1, nil)
	if resp.Error != nil {
		t.Fatalf("tools/list error: %v", resp.Error)
	}
	result := resp.Result.(map[string]any)
	tools := result["tools"].([]mcp.Tool)

	removedTools := map[string]bool{
		"list_datasources": true,
		"list_databases":   true,
		"get_schema":       true,
		"get_table_info":   true,
		"get_view_info":    true,
	}
	for _, tool := range tools {
		if removedTools[tool.Name] {
			t.Errorf("tool %q should have been removed (replaced by resource)", tool.Name)
		}
	}

	// Verify execute_query still exists as a tool.
	found := false
	for _, tool := range tools {
		if tool.Name == "execute_query" {
			found = true
			break
		}
	}
	if !found {
		t.Error("execute_query tool should still exist")
	}

	// Verify execute_query actually works against the real SQLite DB.
	resp = mcpRequest(t, srv, "tools/call", 2, map[string]any{
		"name": "execute_query",
		"arguments": map[string]any{
			"datasource_name": "test-sqlite",
			"database":        "main",
			"sql":             "SELECT id, name FROM users ORDER BY id",
			"limit":           100,
		},
	})
	if resp.Error != nil {
		t.Fatalf("execute_query error: %v", resp.Error)
	}

	toolResult := resp.Result.(map[string]any)
	if toolResult["isError"] == true {
		t.Fatalf("execute_query returned isError: %v", toolResult)
	}
	content := toolResult["content"].([]map[string]any)
	var qr models.QueryResult
	if err := json.Unmarshal([]byte(content[0]["text"].(string)), &qr); err != nil {
		t.Fatalf("invalid query result JSON: %v", err)
	}
	if qr.RowCount != 3 {
		t.Errorf("expected 3 rows, got %d", qr.RowCount)
	}
}
