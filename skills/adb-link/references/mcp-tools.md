# MCP Tool Reference

All tools return JSON strings. Errors return `{"success": false, "error": "..."}`.

## list_datasources

List all configured datasources the current user has permission to access.

- **Parameters**: none
- **Returns**: Array of `{name, type, description, server_info}`

```json
[{"name":"my-pg","type":"postgresql","description":"Production DB","server_info":{"version":"PostgreSQL 16.2"}}]
```

## list_databases

List databases in a datasource.

- **Parameters**:
  - `datasource_name` (string, required) — datasource identifier
- **Returns**: Array of `{name, comment}`

## get_schema

Get complete schema for a database: all tables with columns, types, and comments.

- **Parameters**:
  - `datasource_name` (string, required)
  - `database` (string, required) — database name (index for ES, schema for Oracle)
- **Returns**: `{tables: [{name, columns: [{name, type, comment}]}]}`

## get_table_info

Get detailed column information for a specific table.

- **Parameters**:
  - `datasource_name` (string, required)
  - `database` (string, required)
  - `table` (string, required)
- **Returns**: `{name, columns: [{name, type, comment, nullable, default}]}`

## get_view_info

Get detailed column information for a specific view.

- **Parameters**:
  - `datasource_name` (string, required)
  - `database` (string, required)
  - `view` (string, required)
- **Returns**: Same shape as get_table_info

## execute_query

Execute a query and return structured results.

- **Parameters**:
  - `datasource_name` (string, required)
  - `database` (string, required)
  - `sql` (string, required) — SQL for relational DBs; JSON DSL for ES; command string for Redis; JSON filter/pipeline for MongoDB; JSON query for Milvus
  - `limit` (integer, optional, default: 100) — max rows to return
- **Returns**: `{columns: [string], rows: [[value]], row_count: number}`

## explain_query

Get the execution plan for a SQL statement. Supports MySQL, PostgreSQL, SQLite, ClickHouse, GaussDB, TiDB, MSSQL.

- **Parameters**:
  - `datasource_name` (string, required)
  - `database` (string, required)
  - `sql` (string, required)
- **Returns**: `{success: true, data: <plan rows>}` or `{success: false, error: "unsupported: ..."}`

## submit_async_query

Submit a long-running query for async execution.

- **Parameters**:
  - `datasource_name` (string, required)
  - `database` (string, required)
  - `sql` (string, required)
  - `limit` (integer, optional, default: 100)
  - `timeout_seconds` (integer, optional, default: 300)
- **Returns**: `{success: true, query_id: "..."}`

## submit_async_tool

Submit a registered tool for async execution.

- **Parameters**:
  - `tool_name` (string, required)
  - `parameters` (string, optional, default: "{}") — JSON string of tool parameters
  - `timeout_seconds` (integer, optional, default: 300)
- **Returns**: `{success: true, query_id: "..."}`

## get_async_query_status

Check the status of an async query.

- **Parameters**:
  - `query_id` (string, required)
- **Returns**: `{success: true, data: {status: "running|done|error", ...}}`

## get_async_query_result

Retrieve the result of a completed async query.

- **Parameters**:
  - `query_id` (string, required)
- **Returns**: `{success: true, data: {columns, rows, row_count}}`

## register_datasource

Dynamically register a new datasource. Connection is validated before persisting.

- **Parameters**:
  - `name` (string, required) — unique identifier
  - `type` (string, required) — one of: mysql, postgresql, sqlite, clickhouse, mssql, elasticsearch, hive, gaussdb, redis, mongodb, milvus, oracle, tidb
  - `connection` (object, required) — `{host, port, username, password, default_database, path}`
  - `description` (string, optional)
  - `options` (object, optional) — e.g. `{pool_size: 10}`
- **Returns**: `{success: true, name: "...", persisted_to: "..."}`

## unregister_datasource

Remove a datasource, close connections, delete config file.

- **Parameters**:
  - `name` (string, required)
- **Returns**: `{success: true, name: "..."}`

## register_tool

Create a parameterized query tool with a SQL template.

- **Parameters**:
  - `name` (string, required) — unique tool name
  - `description` (string, required)
  - `datasource` (string, required) — associated datasource
  - `template` (string, required) — SQL/DSL template, use `:param_name` as placeholders
  - `input_schema` (object, required) — JSON Schema defining input parameters
  - `database` (string, optional) — target database
- **Returns**: `{success: true, name: "...", persisted_to: "..."}`

### Example: Register a custom tool

```json
{
  "name": "query_active_users",
  "description": "Query active users created after a given date",
  "datasource": "my-pg",
  "database": "appdb",
  "template": "SELECT id, username, created_at FROM users WHERE active = true AND created_at > :since_date ORDER BY created_at DESC LIMIT :limit",
  "input_schema": {
    "type": "object",
    "properties": {
      "since_date": {"type": "string", "description": "ISO date, e.g. 2024-01-01"},
      "limit": {"type": "integer", "description": "Max rows", "default": 50}
    },
    "required": ["since_date"]
  }
}
```

## unregister_tool

Remove a custom tool and delete its config file.

- **Parameters**:
  - `name` (string, required)
- **Returns**: `{success: true, name: "..."}`
