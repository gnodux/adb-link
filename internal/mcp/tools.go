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
// When accessed via stdio transport (no auth), this returns "" which causes
// permission checks to bypass (anonymous/empty user is treated as bypass).
func userFromCtx(ctx context.Context) string {
	return models.AuthUserNameFromContext(ctx)
}

// RegisterCoreTools attaches the standard set of MCP tools backed by the
// service container.
func RegisterCoreTools(srv *Server, c *services.Container) {
	// list_datasources
	srv.RegisterTool(Tool{
		Name:        "list_datasources",
		Description: "List all configured datasources, including name, type, description, and dialect info.",
		InputSchema: schemaObject(nil, nil),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		user := userFromCtx(ctx)
		all := c.ConfigService.ListDatasources()
		filtered := make([]models.DatasourceInfo, 0, len(all))
		for _, ds := range all {
			if c.PermissionService.CheckDatasource(user, ds.Name) {
				filtered = append(filtered, ds)
			}
		}
		return jsonString(filtered)
	})

	// list_databases
	srv.RegisterTool(Tool{
		Name:        "list_databases",
		Description: "List all databases in a specified datasource, returning name and comment information.",
		InputSchema: schemaObject(map[string]any{
			"datasource_name": prop("string", "Datasource name"),
		}, []string{"datasource_name"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		ds, _ := args["datasource_name"].(string)
		dbs, err := c.SchemaService.GetDatabases(ctx, ds, userFromCtx(ctx))
		if err != nil {
			return "", err
		}
		return jsonString(dbs)
	})

	// get_schema
	srv.RegisterTool(Tool{
		Name:        "get_schema",
		Description: "Get the complete schema of a specified database (tables, columns, types, and comments).",
		InputSchema: schemaObject(map[string]any{
			"datasource_name": prop("string", "Datasource name"),
			"database":        prop("string", "Database name (index name for ES datasources)"),
		}, []string{"datasource_name", "database"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		ds, _ := args["datasource_name"].(string)
		db, _ := args["database"].(string)
		schema, err := c.SchemaService.GetSchema(ctx, ds, db, userFromCtx(ctx))
		if err != nil {
			return "", err
		}
		return jsonString(schema)
	})

	// get_table_info
	srv.RegisterTool(Tool{
		Name:        "get_table_info",
		Description: "Get detailed column information for a specified table.",
		InputSchema: schemaObject(map[string]any{
			"datasource_name": prop("string", "Datasource name"),
			"database":        prop("string", "Database name (index name for ES datasources)"),
			"table":           prop("string", "Table name"),
		}, []string{"datasource_name", "database", "table"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		ds, _ := args["datasource_name"].(string)
		db, _ := args["database"].(string)
		t, _ := args["table"].(string)
		ti, err := c.SchemaService.GetTableInfo(ctx, ds, db, t, userFromCtx(ctx))
		if err != nil {
			return "", err
		}
		return jsonString(ti)
	})

	// get_view_info
	srv.RegisterTool(Tool{
		Name:        "get_view_info",
		Description: "Get detailed column information for a specified view.",
		InputSchema: schemaObject(map[string]any{
			"datasource_name": prop("string", "Datasource name"),
			"database":        prop("string", "Database name (index name for ES datasources)"),
			"view":            prop("string", "View name"),
		}, []string{"datasource_name", "database", "view"}),
	}, func(ctx context.Context, args map[string]any) (string, error) {
		ds, _ := args["datasource_name"].(string)
		db, _ := args["database"].(string)
		v, _ := args["view"].(string)
		vi, err := c.SchemaService.GetViewInfo(ctx, ds, db, v, userFromCtx(ctx))
		if err != nil {
			return "", err
		}
		return jsonString(vi)
	})

	// execute_query
	srv.RegisterTool(Tool{
		Name:        "execute_query",
		Description: "Execute a query on a specified datasource and database, returning structured results.",
		InputSchema: schemaObject(map[string]any{
			"datasource_name": prop("string", "Datasource name"),
			"database":        prop("string", "Database name (index name for ES datasources)"),
			"sql":             prop("string", "SQL statement; for ES datasources, provide a JSON DSL query body"),
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
		Description: "Get the execution plan for a SQL statement. Supports MySQL, PostgreSQL, SQLite, ClickHouse, and MSSQL.",
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
