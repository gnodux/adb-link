---
title: MCP Tools
---

# MCP Tools Reference

ADB-Link implements the [Model Context Protocol (MCP)](https://modelcontextprotocol.io) with full JSON-RPC 2.0 support. All tools are available via `tools/call` method.

## Transport Modes

| Mode | Command | Use Case |
|------|---------|----------|
| stdio | `adb-link run-mcp` | IDE/agent integration (Claude Desktop, Cursor) |
| HTTP | `adb-link run-all` | Remote access, multi-client |

---

## Tool List

### list_datasources

List all available datasources.

**Arguments:** none

**Response:**
```json
[
  {
    "name": "my-postgres",
    "type": "postgresql",
    "description": "Production PostgreSQL"
  }
]
```

---

### list_databases

List databases in a datasource.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `datasource_name` | string | Yes | Target datasource |

**Response:**
```json
[
  {"name": "mydb", "comment": "Main database"},
  {"name": "analytics", "comment": ""}
]
```

---

### get_schema

Get full schema (tables and columns) for a database.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `datasource_name` | string | Yes | Target datasource |
| `database` | string | Yes | Target database |

---

### get_table_info

Get detailed column information for a specific table.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `datasource_name` | string | Yes | Target datasource |
| `database` | string | Yes | Target database |
| `table` | string | Yes | Table name |

---

### get_view_info

Get detailed column information for a specific view.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `datasource_name` | string | Yes | Target datasource |
| `database` | string | Yes | Target database |
| `view` | string | Yes | View name |

---

### execute_query

Execute a SQL or DSL query.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `datasource_name` | string | Yes | Target datasource |
| `database` | string | Yes | Target database |
| `sql` | string | Yes | SQL/DSL query string |

**Response:**
```json
{
  "columns": ["id", "username", "created_at"],
  "rows": [[1, "alice", "2026-03-15"]],
  "row_count": 1
}
```

---

### explain_query

Get the execution plan for a query.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `datasource_name` | string | Yes | Target datasource |
| `database` | string | Yes | Target database |
| `sql` | string | Yes | SQL query to explain |

---

### submit_async_query

Submit a long-running query for async execution.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `datasource_name` | string | Yes | Target datasource |
| `database` | string | Yes | Target database |
| `sql` | string | Yes | SQL query |

**Response:**
```json
{
  "query_id": "abc123-def456"
}
```

---

### get_async_query_status

Check the status of an async query.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query_id` | string | Yes | Async query ID |

**Response:**
```json
{
  "query_id": "abc123-def456",
  "status": "completed"
}
```

Status values: `pending`, `running`, `completed`, `failed`, `cancelled`

---

### get_async_query_result

Retrieve the result of a completed async query.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query_id` | string | Yes | Async query ID |

---

### register_tool

Register a dynamic query tool at runtime.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | Tool name |
| `description` | string | Yes | Tool description |
| `datasource` | string | Yes | Target datasource |
| `database` | string | Yes | Target database |
| `sql` | string | Yes | SQL template with `:param` placeholders |
| `parameters` | array | Yes | Parameter definitions |

---

### unregister_tool

Remove a dynamic tool.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `tool_name` | string | Yes | Tool name to remove |

---

### register_datasource

Register a new datasource at runtime with connection validation.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | Datasource name |
| `type` | string | Yes | Database type |
| `connection` | object | Yes | Connection details |

---

### unregister_datasource

Remove a datasource at runtime.

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `datasource_name` | string | Yes | Datasource name to remove |

---

## Dynamic Tools

Any tool registered via `register_tool` (API or MCP) becomes immediately available as a new MCP tool. Dynamic tools appear in `tools/list` and can be called like built-in tools.

Example -- register and call a custom tool:

```json
// 1. Register
{
  "jsonrpc": "2.0", "id": 1, "method": "tools/call",
  "params": {
    "name": "register_tool",
    "arguments": {
      "name": "get_active_users",
      "description": "Get active users from the last N days",
      "datasource": "my-postgres",
      "database": "mydb",
      "sql": "SELECT * FROM users WHERE last_active > NOW() - INTERVAL ':days days'",
      "parameters": [
        {"name": "days", "type": "integer", "required": true, "description": "Number of days"}
      ]
    }
  }
}

// 2. Call the new tool
{
  "jsonrpc": "2.0", "id": 2, "method": "tools/call",
  "params": {
    "name": "get_active_users",
    "arguments": {"days": 7}
  }
}
```

---

## Permission Enforcement

MCP tools respect the same RBAC permissions as the REST API. The MCP stdio transport uses `mcp_stdio_user` as the default identity. Ensure this user has appropriate permissions configured.

Empty/anonymous users bypass permission checks (useful for local development).
