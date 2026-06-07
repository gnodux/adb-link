package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gnodux/adb-link/internal/models"
	"github.com/gnodux/adb-link/internal/services"
)

// userFromCtx extracts the authenticated user name from the context.
// When accessed via HTTP transport, the BearerAuth middleware injects the user.
// When accessed via stdio transport, the context carries "mcp_stdio_user" so
// that permission rules can be applied consistently.
func userFromCtx(ctx context.Context) string {
	return models.AuthUserNameFromContext(ctx)
}

// RegisterCoreTools attaches the standard set of MCP tools backed by the
// service container.
func RegisterCoreTools(srv *Server, c *services.Container) {
	// Read-only datasource tools (list_datasources, list_databases, get_schema,
	// get_table_info, get_view_info) have been migrated to MCP Resources.
	// See resources.go for the new resource-based equivalents.

	// execute_query
	srv.RegisterTool(Tool{
		Name:        "execute_query",
		Description: "Execute a query on a specified datasource and database, returning structured results.",
		InputSchema: schemaObject(map[string]any{
			"datasource_name": prop("string", "Datasource name"),
			"database":        prop("string", "Database name (index for ES, schema for Oracle, database for MongoDB/Milvus/GaussDB/TiDB, ignored for Redis)"),
			"sql":             prop("string", "Query statement: SQL for relational DBs; JSON DSL for Elasticsearch; Redis command string; JSON filter/pipeline for MongoDB; JSON query/search for Milvus"),
			"limit":           propWithDefault("integer", "Maximum number of rows to return", 100),
		}, []string{"datasource_name", "database", "sql"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		req := &models.QueryRequest{
			DatasourceName: stringArg(args, "datasource_name"),
			Database:       stringArg(args, "database"),
			SQL:            stringArg(args, "sql"),
			Limit:          intArg(args, "limit", 100),
			TimeoutSeconds: 30,
		}
		result, err := c.QueryService.Execute(ctx, req, userFromCtx(ctx))
		if err != nil {
			return "", err
		}
		return jsonString(result)
	})

	// explain_query
	srv.RegisterTool(Tool{
		Name:        "explain_query",
		Description: "Get the execution plan for a SQL statement. Supports MySQL, PostgreSQL, SQLite, ClickHouse, GaussDB, TiDB, and MSSQL.",
		InputSchema: schemaObject(map[string]any{
			"datasource_name": prop("string", "Datasource name"),
			"database":        prop("string", "Database name"),
			"sql":             prop("string", "SQL statement"),
		}, []string{"datasource_name", "database", "sql"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		req := &models.ExplainRequest{
			DatasourceName: stringArg(args, "datasource_name"),
			Database:       stringArg(args, "database"),
			SQL:            stringArg(args, "sql"),
			TimeoutSeconds: 30,
		}
		result, err := c.QueryService.Explain(ctx, req, userFromCtx(ctx))
		if err != nil {
			if _, ok := err.(*services.UnsupportedOperationError); ok {
				return jsonString(map[string]any{"success": false, "error": "unsupported: " + err.Error()})
			}
			return "", err
		}
		return jsonString(map[string]any{"success": true, "data": result})
	})

	// submit_async_query
	srv.RegisterTool(Tool{
		Name:        "submit_async_query",
		Description: "Submit an async query task, returns a query_id for polling status and retrieving results.",
		InputSchema: schemaObject(map[string]any{
			"datasource_name": prop("string", "Datasource name"),
			"database":        prop("string", "Database name"),
			"sql":             prop("string", "SQL query statement"),
			"limit":           propWithDefault("integer", "Maximum number of rows to return", 100),
			"timeout_seconds": propWithDefault("integer", "Query timeout in seconds", 300),
		}, []string{"datasource_name", "database", "sql"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		ds := stringArg(args, "datasource_name")
		cfg, err := c.ConfigService.GetDatasource(ds)
		if err != nil {
			return "", err
		}
		if cfg.Shadow {
			return jsonString(map[string]any{"success": false, "error": "shadow数据源不允许直接查询"})
		}
		req := &models.AsyncQueryRequest{
			DatasourceName: ds,
			Database:       stringArg(args, "database"),
			SQL:            stringArg(args, "sql"),
			Limit:          intArg(args, "limit", 100),
			TimeoutSeconds: intArg(args, "timeout_seconds", 300),
		}
		queryID, err := c.AsyncQueryService.Submit(req, userFromCtx(ctx))
		if err != nil {
			return "", err
		}
		return jsonString(map[string]any{"success": true, "query_id": queryID})
	})

	// submit_async_tool
	srv.RegisterTool(Tool{
		Name:        "submit_async_tool",
		Description: "Submit a tool for async execution, returns a query_id.",
		InputSchema: schemaObject(map[string]any{
			"tool_name":       prop("string", "Tool name"),
			"parameters":      propWithDefault("string", "Tool parameters as JSON string", "{}"),
			"timeout_seconds": propWithDefault("integer", "Execution timeout in seconds", 300),
		}, []string{"tool_name"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		toolName := stringArg(args, "tool_name")
		paramsStr := stringArg(args, "parameters")
		if paramsStr == "" {
			paramsStr = "{}"
		}
		var params map[string]any
		if err := json.Unmarshal([]byte(paramsStr), &params); err != nil {
			return jsonString(map[string]any{"success": false, "error": fmt.Sprintf("Invalid JSON parameters: %s", err)})
		}
		queryID, err := c.AsyncQueryService.SubmitTool(toolName, params, intArg(args, "timeout_seconds", 300), userFromCtx(ctx))
		if err != nil {
			return jsonString(map[string]any{"success": false, "error": err.Error()})
		}
		return jsonString(map[string]any{"success": true, "query_id": queryID})
	})

	// get_async_query_status
	srv.RegisterTool(Tool{
		Name:        "get_async_query_status",
		Description: "Get the execution status of an async query.",
		InputSchema: schemaObject(map[string]any{
			"query_id": prop("string", "Async query ID"),
		}, []string{"query_id"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		status, err := c.AsyncQueryService.GetStatus(stringArg(args, "query_id"))
		if err != nil {
			return jsonString(map[string]any{"success": false, "error": err.Error()})
		}
		return jsonString(map[string]any{"success": true, "data": status})
	})

	// get_async_query_result
	srv.RegisterTool(Tool{
		Name:        "get_async_query_result",
		Description: "Get the result of an async query. Only available after the query completes.",
		InputSchema: schemaObject(map[string]any{
			"query_id": prop("string", "Async query ID"),
		}, []string{"query_id"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		result, err := c.AsyncQueryService.GetResult(stringArg(args, "query_id"))
		if err != nil {
			return jsonString(map[string]any{"success": false, "error": err.Error()})
		}
		return jsonString(map[string]any{"success": true, "data": result})
	})

	// register_tool
	srv.RegisterTool(Tool{
		Name:        "register_tool",
		Description: "Dynamically register a new query tool. The tool becomes available immediately and is persisted.",
		InputSchema: schemaObject(map[string]any{
			"name":         prop("string", "Tool name (unique identifier)"),
			"description":  prop("string", "Tool description"),
			"datasource":   prop("string", "Associated datasource name"),
			"template":     prop("string", "SQL/DSL template, use :param_name as parameter placeholders"),
			"database":     propWithDefault("string", "Target database", ""),
			"input_schema": prop("object", "JSON Schema object defining the tool's input parameters (type, properties, required)"),
		}, []string{"name", "description", "datasource", "template", "input_schema"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		name := stringArg(args, "name")
		if _, err := c.ConfigService.GetTool(name); err == nil {
			return jsonString(map[string]any{"success": false, "error": "Tool '" + name + "' already exists"})
		}
		// Parse input_schema — accept either a JSON string or a map
		var schemaBytes []byte
		switch v := args["input_schema"].(type) {
		case string:
			schemaBytes = []byte(v)
		case map[string]any:
			b, err := json.Marshal(v)
			if err != nil {
				return jsonString(map[string]any{"success": false, "error": "Invalid input_schema: " + err.Error()})
			}
			schemaBytes = b
		default:
			return jsonString(map[string]any{"success": false, "error": "input_schema must be a JSON Schema object"})
		}
		// Validate it's a valid JSON object
		var schemaMap map[string]any
		if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
			return jsonString(map[string]any{"success": false, "error": "Invalid input_schema JSON: " + err.Error()})
		}
		tool := &models.ToolConfig{
			Kind:        "tool",
			Name:        name,
			Description: stringArg(args, "description"),
			Datasource:  stringArg(args, "datasource"),
			Database:    stringArg(args, "database"),
			Template:    stringArg(args, "template"),
			InputSchema: schemaBytes,
		}
		c.ConfigService.RegisterTool(tool)
		RegisterDynamicTool(srv, c, tool)
		filePath, err := c.ConfigService.PersistTool(tool)
		if err != nil {
			return "", err
		}
		services.AuditLog().Printf("user=%s | action=register_tool | tool=%s", userFromCtx(ctx), name)
		return jsonString(map[string]any{"success": true, "name": name, "persisted_to": filePath})
	})

	// unregister_tool
	srv.RegisterTool(Tool{
		Name:        "unregister_tool",
		Description: "Unregister a tool. Removes it from memory and deletes the config file.",
		InputSchema: schemaObject(map[string]any{
			"name": prop("string", "Name of the tool to unregister"),
		}, []string{"name"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		name := stringArg(args, "name")
		if _, err := c.ConfigService.GetTool(name); err != nil {
			return jsonString(map[string]any{"success": false, "error": "Tool '" + name + "' not found"})
		}
		c.ConfigService.UnregisterTool(name)
		srv.UnregisterTool(name)
		c.ConfigService.RemoveToolFile(name)
		services.AuditLog().Printf("user=%s | action=unregister_tool | tool=%s", userFromCtx(ctx), name)
		return jsonString(map[string]any{"success": true, "name": name})
	})

	// register_datasource
	srv.RegisterTool(Tool{
		Name:        "register_datasource",
		Description: "Dynamically register a new datasource. The connection is validated before registration. The datasource is persisted to config.",
		InputSchema: schemaObject(map[string]any{
			"name":        prop("string", "Datasource name (unique identifier)"),
			"type":        prop("string", "Database type: mysql, postgresql, sqlite, clickhouse, mssql, elasticsearch, hive, gaussdb, redis, mongodb, milvus, oracle, tidb"),
			"description": propWithDefault("string", "Datasource description", ""),
			"connection":  prop("object", "Connection config: {host, port, username, password, default_database, path}"),
			"options":     prop("object", "Additional options (pool_size, etc.)"),
		}, []string{"name", "type", "connection"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		name := stringArg(args, "name")
		if name == "" {
			return jsonString(map[string]any{"success": false, "error": "name is required"})
		}
		if _, err := c.ConfigService.GetDatasource(name); err == nil {
			return jsonString(map[string]any{"success": false, "error": "Datasource '" + name + "' already exists"})
		}
		var conn models.ConnectionConfig
		switch v := args["connection"].(type) {
		case map[string]any:
			b, _ := json.Marshal(v)
			_ = json.Unmarshal(b, &conn)
		case string:
			_ = json.Unmarshal([]byte(v), &conn)
		default:
			return jsonString(map[string]any{"success": false, "error": "connection must be an object"})
		}
		var options map[string]any
		if v, ok := args["options"].(map[string]any); ok {
			options = v
		}
		cfg := &models.DatasourceConfig{
			Kind:        "datasource",
			Name:        name,
			Type:        models.DatabaseType(stringArg(args, "type")),
			Description: stringArg(args, "description"),
			Connection:  conn,
			Options:     options,
		}
		c.ConfigService.RegisterDatasource(cfg)
		if services.IsNonSQLType(cfg.Type) {
			client, _, err := c.ConnectionService.GetNonSQLClient(name)
			if err != nil {
				c.ConfigService.UnregisterDatasource(name)
				return jsonString(map[string]any{"success": false, "error": "Connection validation failed: " + err.Error()})
			}
			if err := client.Ping(ctx); err != nil {
				c.ConfigService.UnregisterDatasource(name)
				c.ConnectionService.Invalidate(name)
				return jsonString(map[string]any{"success": false, "error": "Connection validation failed: " + err.Error()})
			}
		} else {
			db, _, err := c.ConnectionService.GetSQLDB(name, "")
			if err != nil {
				c.ConfigService.UnregisterDatasource(name)
				return jsonString(map[string]any{"success": false, "error": "Connection validation failed: " + err.Error()})
			}
			if err := db.PingContext(ctx); err != nil {
				c.ConfigService.UnregisterDatasource(name)
				c.ConnectionService.Invalidate(name)
				return jsonString(map[string]any{"success": false, "error": "Connection validation failed: " + err.Error()})
			}
		}
		c.ConnectionService.Invalidate(name)
		filePath, err := c.ConfigService.PersistDatasource(cfg)
		if err != nil {
			return jsonString(map[string]any{"success": false, "error": err.Error()})
		}
		services.AuditLog().Printf("user=%s | action=register_datasource | datasource=%s | type=%s",
			userFromCtx(ctx), name, cfg.Type)
		return jsonString(map[string]any{"success": true, "name": name, "persisted_to": filePath})
	})

	// unregister_datasource
	srv.RegisterTool(Tool{
		Name:        "unregister_datasource",
		Description: "Unregister a datasource. Removes it from memory, closes connections, and deletes the config file.",
		InputSchema: schemaObject(map[string]any{
			"name": prop("string", "Name of the datasource to unregister"),
		}, []string{"name"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		name := stringArg(args, "name")
		if _, err := c.ConfigService.GetDatasource(name); err != nil {
			return jsonString(map[string]any{"success": false, "error": "Datasource '" + name + "' not found"})
		}
		c.ConfigService.UnregisterDatasource(name)
		c.ConnectionService.Invalidate(name)
		c.ConfigService.RemoveDatasourceFile(name)
		services.AuditLog().Printf("user=%s | action=unregister_datasource | datasource=%s", userFromCtx(ctx), name)
		return jsonString(map[string]any{"success": true, "name": name})
	})
}

// RegisterDynamicTools registers all YAML-configured tools as MCP tools.
func RegisterDynamicTools(srv *Server, c *services.Container) {
	for _, tool := range c.ConfigService.AllTools() {
		RegisterDynamicTool(srv, c, tool)
	}
}

// RegisterDynamicTool registers a single dynamic tool against the MCP server.
func RegisterDynamicTool(srv *Server, c *services.Container, tool *models.ToolConfig) {
	tc := tool
	srv.RegisterTool(Tool{
		Name:        tc.Name,
		Description: tc.Description,
		InputSchema: tc.BuildInputSchema(),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		user := userFromCtx(ctx)
		if !c.PermissionService.CheckTool(user, tc.Name) {
			return jsonString(map[string]any{"success": false, "error": "Access denied: user '" + user + "' cannot execute tool '" + tc.Name + "'"})
		}
		result, err := c.QueryService.ExecuteTemplate(ctx, tc, args, user)
		if err != nil {
			return jsonString(map[string]any{"success": false, "error": "Error executing tool '" + tc.Name + "': " + err.Error()})
		}
		return jsonString(map[string]any{
			"columns":   result.Columns,
			"rows":      result.Rows,
			"row_count": result.RowCount,
		})
	})
}

// helpers

func schemaObject(props map[string]any, required []string) map[string]any {
	if props == nil {
		props = map[string]any{}
	}
	out := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func prop(t, desc string) map[string]any {
	return map[string]any{"type": t, "description": desc}
}

func propWithDefault(t, desc string, def any) map[string]any {
	return map[string]any{"type": t, "description": desc, "default": def}
}

func stringArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func intArg(args map[string]any, key string, def int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return def
}

func jsonString(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
