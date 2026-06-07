//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gnodux/adb-link/internal/config"
	"github.com/gnodux/adb-link/internal/mcp"
	"github.com/gnodux/adb-link/internal/models"
	"github.com/gnodux/adb-link/internal/services"
	"github.com/gnodux/adb-link/tests/testutil"
	_ "github.com/go-sql-driver/mysql"
)

// setupMySQLMCP creates a real MySQL container, seeds it, wires up a full
// services.Container and MCP server, and returns them for testing.
func setupMySQLMCP(t *testing.T) (*mcp.Server, *services.Container) {
	t.Helper()

	container := testutil.StartContainer(t, "mysql:8", 3306,
		[]string{
			"MYSQL_ROOT_PASSWORD=test",
			"MYSQL_DATABASE=testdb",
		}, nil)

	dsn := fmt.Sprintf("root:test@tcp(localhost:%s)/testdb?charset=utf8mb4&parseTime=true&loc=Local&timeout=5s&readTimeout=60s&writeTimeout=60s", container.HostPort)
	testutil.WaitForSQL(t, "mysql", dsn, 120*time.Second)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	seedMySQL(t, db)

	dir := t.TempDir()
	writeYAML(t, dir, "ds.yaml", fmt.Sprintf(`
kind: datasource
name: test-mysql
type: mysql
connection:
  host: localhost
  port: %s
  username: root
  password: test
  default_database: testdb
`, container.HostPort))

	cs := config.NewConfigService(&config.Settings{ConfigDir: dir})
	conn := services.NewConnectionService(cs)
	t.Cleanup(func() { conn.DisposeAll() })

	metaSvc := services.NewMetadataService(cs.AllMetadata())
	permSvc := services.NewPermissionService(nil, nil)
	schemaSvc := services.NewSchemaService(cs, conn, metaSvc, permSvc)
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

	srv := mcp.NewServer("adb-link-test", "0.0.1")
	mcp.RegisterCoreTools(srv, c)
	mcp.RegisterDynamicTools(srv, c)
	mcp.RegisterCoreResources(srv, c)

	return srv, c
}

// mcpCall is a helper to make a JSON-RPC request against the MCP server with
// an empty user (bypass permissions).
func mcpCall(t *testing.T, srv *mcp.Server, method string, id int, params any) *mcp.Response {
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
	ctx = models.WithAuthUser(ctx, &models.AuthUser{Name: ""})
	return srv.HandleRequest(ctx, req)
}

// --- Integration Tests ---

func TestMySQL_MCP_Initialize_HasResourcesCapability(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "initialize", 1, nil)
	if resp.Error != nil {
		t.Fatalf("initialize error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	caps := result["capabilities"].(map[string]any)
	resCap, ok := caps["resources"].(map[string]any)
	if !ok {
		t.Fatal("missing resources capability")
	}
	if resCap["listChanged"] != true {
		t.Fatalf("expected resources.listChanged=true, got %v", resCap["listChanged"])
	}
	toolsCap, ok := caps["tools"].(map[string]any)
	if !ok {
		t.Fatal("missing tools capability")
	}
	if toolsCap["listChanged"] != true {
		t.Fatalf("expected tools.listChanged=true, got %v", toolsCap["listChanged"])
	}
}

func TestMySQL_MCP_ResourcesList(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "resources/list", 1, nil)
	if resp.Error != nil {
		t.Fatalf("resources/list error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	resources, ok := result["resources"].([]mcp.Resource)
	if !ok {
		t.Fatalf("expected []Resource, got %T", result["resources"])
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 static resource, got %d", len(resources))
	}
	r := resources[0]
	if r.URI != "datasource:///" {
		t.Errorf("expected URI %q, got %q", "datasource:///", r.URI)
	}
	if r.MimeType != "application/json" {
		t.Errorf("expected mimeType %q, got %q", "application/json", r.MimeType)
	}
}

func TestMySQL_MCP_TemplatesList(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "resources/templates/list", 1, nil)
	if resp.Error != nil {
		t.Fatalf("resources/templates/list error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	templates, ok := result["resourceTemplates"].([]mcp.ResourceTemplate)
	if !ok {
		t.Fatalf("expected []ResourceTemplate, got %T", result["resourceTemplates"])
	}
	if len(templates) != 5 {
		t.Fatalf("expected 5 templates, got %d", len(templates))
	}

	expectedURIs := map[string]bool{
		"datasource:///{name}":                     false,
		"datasource:///{name}/{db}/databases":      false,
		"datasource:///{name}/{db}/schema":         false,
		"datasource:///{name}/{db}/tables/{table}": false,
		"datasource:///{name}/{db}/views/{view}":   false,
	}
	for _, tmpl := range templates {
		if _, ok := expectedURIs[tmpl.URITemplate]; ok {
			expectedURIs[tmpl.URITemplate] = true
		}
	}
	for uri, found := range expectedURIs {
		if !found {
			t.Errorf("missing template: %s", uri)
		}
	}
}

func TestMySQL_MCP_Read_DatasourceList(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "resources/read", 1, map[string]any{
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
	ds := datasources[0]
	if ds.Name != "test-mysql" {
		t.Errorf("expected name %q, got %q", "test-mysql", ds.Name)
	}
	if string(ds.Type) != "mysql" {
		t.Errorf("expected type %q, got %q", "mysql", ds.Type)
	}
}

func TestMySQL_MCP_Read_DatasourceDetail(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-mysql",
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
	if cfg.Name != "test-mysql" {
		t.Errorf("expected name %q, got %q", "test-mysql", cfg.Name)
	}
	if string(cfg.Type) != "mysql" {
		t.Errorf("expected type %q, got %q", "mysql", cfg.Type)
	}
	if cfg.Connection.Host != "localhost" {
		t.Errorf("expected host %q, got %q", "localhost", cfg.Connection.Host)
	}
	if cfg.Connection.Username != "root" {
		t.Errorf("expected username %q, got %q", "root", cfg.Connection.Username)
	}
	// Password should be masked.
	if cfg.Connection.Password != "***" {
		t.Errorf("expected masked password %q, got %q", "***", cfg.Connection.Password)
	}
	if cfg.Connection.DefaultDatabase != "testdb" {
		t.Errorf("expected default_database %q, got %q", "testdb", cfg.Connection.DefaultDatabase)
	}
}

func TestMySQL_MCP_Read_Databases(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-mysql/testdb/databases",
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

	// MySQL 8 creates several system databases; we at minimum need testdb.
	found := false
	for _, d := range dbs {
		if d.Name == "testdb" {
			found = true
			break
		}
	}
	if !found {
		names := make([]string, len(dbs))
		for i, d := range dbs {
			names[i] = d.Name
		}
		t.Errorf("expected testdb in databases, got %v", names)
	}
}

func TestMySQL_MCP_Read_Schema(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-mysql/testdb/schema",
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

	tableNames := map[string]bool{}
	for _, tbl := range schema.Tables {
		tableNames[tbl.Name] = true
	}
	if !tableNames["users"] {
		t.Error("missing table 'users'")
	}
	if !tableNames["orders"] {
		t.Error("missing table 'orders'")
	}

	viewNames := map[string]bool{}
	for _, v := range schema.Views {
		viewNames[v.Name] = true
	}
	if !viewNames["user_order_summary"] {
		t.Error("missing view 'user_order_summary'")
	}
}

func TestMySQL_MCP_Read_TableInfo(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-mysql/testdb/tables/users",
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

	colNames := map[string]bool{}
	for _, col := range ti.Columns {
		colNames[col.Name] = true
	}
	for _, expected := range []string{"id", "name", "email", "created_at"} {
		if !colNames[expected] {
			t.Errorf("missing column %q", expected)
		}
	}

	// Verify primary key detection.
	foundPK := false
	for _, col := range ti.Columns {
		if col.Name == "id" && col.IsPrimaryKey {
			foundPK = true
		}
	}
	if !foundPK {
		t.Error("expected 'id' to be marked as primary key")
	}
}

func TestMySQL_MCP_Read_ViewInfo(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-mysql/testdb/views/user_order_summary",
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
		t.Errorf("expected 3 columns, got %d", len(vi.Columns))
	}

	colNames := map[string]bool{}
	for _, col := range vi.Columns {
		colNames[col.Name] = true
	}
	for _, expected := range []string{"name", "order_count", "total"} {
		if !colNames[expected] {
			t.Errorf("missing view column %q", expected)
		}
	}
}

func TestMySQL_MCP_Read_OrdersTable(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-mysql/testdb/tables/orders",
	})
	if resp.Error != nil {
		t.Fatalf("resources/read error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	contents := result["contents"].([]mcp.ResourceContent)

	var ti models.TableInfo
	if err := json.Unmarshal([]byte(contents[0].Text), &ti); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if ti.Name != "orders" {
		t.Errorf("expected table name %q, got %q", "orders", ti.Name)
	}
	if len(ti.Columns) != 4 {
		t.Errorf("expected 4 columns, got %d", len(ti.Columns))
	}

	colNames := map[string]bool{}
	for _, col := range ti.Columns {
		colNames[col.Name] = true
	}
	for _, expected := range []string{"id", "user_id", "amount", "status"} {
		if !colNames[expected] {
			t.Errorf("missing column %q", expected)
		}
	}
}

func TestMySQL_MCP_Read_NonexistentDatasource(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///does-not-exist",
	})
	if resp.Error == nil {
		t.Fatal("expected error for nonexistent datasource, got nil")
	}
}

func TestMySQL_MCP_Read_InvalidURI(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-mysql/testdb/unknown-segment",
	})
	if resp.Error == nil {
		t.Fatal("expected error for invalid URI, got nil")
	}
}

func TestMySQL_MCP_Read_NonexistentTable(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-mysql/testdb/tables/nonexistent_table",
	})
	if resp.Error != nil {
		// MySQL dialect returns an error — that's acceptable.
		return
	}
	// Or it returns an empty TableInfo (no columns) — also acceptable.
	result := resp.Result.(map[string]any)
	contents := result["contents"].([]mcp.ResourceContent)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}
	var ti models.TableInfo
	if err := json.Unmarshal([]byte(contents[0].Text), &ti); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(ti.Columns) != 0 {
		t.Errorf("expected 0 columns for nonexistent table, got %d", len(ti.Columns))
	}
}

func TestMySQL_MCP_ToolsStillWork(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	// Verify execute_query tool is still registered.
	resp := mcpCall(t, srv, "tools/list", 1, nil)
	if resp.Error != nil {
		t.Fatalf("tools/list error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	tools := result["tools"].([]mcp.Tool)

	// Read-only tools should be gone.
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

	// execute_query should still be present and functional.
	found := false
	for _, tool := range tools {
		if tool.Name == "execute_query" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("execute_query tool should still exist")
	}

	// Actually run a query against the real MySQL.
	resp = mcpCall(t, srv, "tools/call", 2, map[string]any{
		"name": "execute_query",
		"arguments": map[string]any{
			"datasource_name": "test-mysql",
			"database":        "testdb",
			"sql":             "SELECT id, name, email FROM users ORDER BY id",
			"limit":           100,
		},
	})
	if resp.Error != nil {
		t.Fatalf("execute_query RPC error: %v", resp.Error)
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
	if len(qr.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(qr.Columns))
	}
}

func TestMySQL_MCP_ExplainQueryTool(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	resp := mcpCall(t, srv, "tools/call", 1, map[string]any{
		"name": "explain_query",
		"arguments": map[string]any{
			"datasource_name": "test-mysql",
			"database":        "testdb",
			"sql":             "SELECT * FROM users WHERE id = 1",
		},
	})
	if resp.Error != nil {
		t.Fatalf("explain_query RPC error: %v", resp.Error)
	}

	toolResult := resp.Result.(map[string]any)
	if toolResult["isError"] == true {
		t.Fatalf("explain_query returned isError: %v", toolResult)
	}
	content := toolResult["content"].([]map[string]any)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content[0]["text"].(string)), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed["success"] != true {
		t.Errorf("expected success=true, got %v", parsed["success"])
	}
}

func TestMySQL_MCP_ResourceAndToolCoexistence(t *testing.T) {
	srv, _ := setupMySQLMCP(t)

	// 1. Read table schema via resource.
	resp := mcpCall(t, srv, "resources/read", 1, map[string]any{
		"uri": "datasource:///test-mysql/testdb/tables/users",
	})
	if resp.Error != nil {
		t.Fatalf("resources/read error: %v", resp.Error)
	}
	resResult := resp.Result.(map[string]any)
	resContents := resResult["contents"].([]mcp.ResourceContent)
	var ti models.TableInfo
	if err := json.Unmarshal([]byte(resContents[0].Text), &ti); err != nil {
		t.Fatalf("invalid resource JSON: %v", err)
	}
	if ti.Name != "users" {
		t.Errorf("resource: expected name %q, got %q", "users", ti.Name)
	}
	if len(ti.Columns) != 4 {
		t.Errorf("resource: expected 4 columns, got %d", len(ti.Columns))
	}

	// 2. Execute a query via tool against the same table.
	resp = mcpCall(t, srv, "tools/call", 2, map[string]any{
		"name": "execute_query",
		"arguments": map[string]any{
			"datasource_name": "test-mysql",
			"database":        "testdb",
			"sql":             "SELECT name FROM users ORDER BY id",
			"limit":           10,
		},
	})
	if resp.Error != nil {
		t.Fatalf("execute_query error: %v", resp.Error)
	}
	toolResult := resp.Result.(map[string]any)
	if toolResult["isError"] == true {
		t.Fatalf("execute_query isError: %v", toolResult)
	}
	toolContent := toolResult["content"].([]map[string]any)
	var qr models.QueryResult
	if err := json.Unmarshal([]byte(toolContent[0]["text"].(string)), &qr); err != nil {
		t.Fatalf("invalid query JSON: %v", err)
	}
	if qr.RowCount != 3 {
		t.Errorf("tool: expected 3 rows, got %d", qr.RowCount)
	}
}
