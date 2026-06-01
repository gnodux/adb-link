---
title: API Reference
---

# REST API Reference

All API endpoints require Bearer token authentication (except `/api/health`).

```
Authorization: Bearer <your-api-key>
```

Base URL: `http://localhost:8000`

---

## Health Check

### GET /api/health

Liveness check. No authentication required.

**Response:**
```json
{"status": "ok"}
```

---

## Datasources

### POST /api/datasources/list

List all configured datasources.

**Request Body:**
```json
{}
```

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

### POST /api/datasources/detail

Get detailed information about a datasource.

**Request Body:**
```json
{
  "datasource_name": "my-postgres"
}
```

### POST /api/datasources/test

Test connectivity to a datasource.

**Request Body:**
```json
{
  "datasource_name": "my-postgres"
}
```

**Response:**
```json
{
  "status": "ok",
  "message": "Connection successful"
}
```

### POST /api/datasources/register

Register a new datasource at runtime (no restart required).

**Request Body:**
```json
{
  "name": "new-db",
  "type": "mysql",
  "connection": {
    "host": "127.0.0.1",
    "port": 3306,
    "username": "root",
    "password": "secret",
    "default_database": "mydb"
  }
}
```

### POST /api/datasources/unregister

Remove a datasource at runtime.

**Request Body:**
```json
{
  "datasource_name": "new-db"
}
```

---

## Databases

### POST /api/databases/list

List databases available in a datasource.

**Request Body:**
```json
{
  "datasource_name": "my-postgres"
}
```

**Response:**
```json
[
  {"name": "mydb", "comment": "Main application database"},
  {"name": "analytics", "comment": ""}
]
```

---

## Schema

### POST /api/schema/get

Get full schema (tables and columns) for a database.

**Request Body:**
```json
{
  "datasource_name": "my-postgres",
  "database": "mydb"
}
```

**Response:**
```json
{
  "tables": [
    {
      "name": "users",
      "columns": [
        {"name": "id", "type": "INT4", "nullable": false, "comment": "Primary key"},
        {"name": "username", "type": "VARCHAR", "nullable": false, "comment": ""}
      ]
    }
  ]
}
```

### POST /api/schema/table

Get detailed information about a specific table.

**Request Body:**
```json
{
  "datasource_name": "my-postgres",
  "database": "mydb",
  "table": "users"
}
```

### POST /api/schema/view

Get detailed information about a specific view.

**Request Body:**
```json
{
  "datasource_name": "my-postgres",
  "database": "mydb",
  "view": "active_users_view"
}
```

---

## Query

### POST /api/query/execute

Execute a SQL query.

**Request Body:**
```json
{
  "datasource_name": "my-postgres",
  "database": "mydb",
  "sql": "SELECT id, username FROM users LIMIT 10"
}
```

**Response:**
```json
{
  "columns": ["id", "username"],
  "rows": [[1, "alice"], [2, "bob"]],
  "row_count": 2
}
```

### POST /api/query/explain

Get the execution plan for a query.

**Request Body:**
```json
{
  "datasource_name": "my-postgres",
  "database": "mydb",
  "sql": "SELECT * FROM users WHERE id = 1"
}
```

---

## Async Query

### POST /api/async/query/submit

Submit a long-running query for async execution.

**Request Body:**
```json
{
  "datasource_name": "my-postgres",
  "database": "mydb",
  "sql": "SELECT * FROM large_table"
}
```

**Response:**
```json
{
  "query_id": "abc123-def456"
}
```

### POST /api/async/query/status

Check the status of an async query.

**Request Body:**
```json
{
  "query_id": "abc123-def456"
}
```

**Response:**
```json
{
  "query_id": "abc123-def456",
  "status": "completed"
}
```

### POST /api/async/query/result

Retrieve the result of a completed async query.

**Request Body:**
```json
{
  "query_id": "abc123-def456"
}
```

### POST /api/async/query/cancel

Cancel a running async query.

**Request Body:**
```json
{
  "query_id": "abc123-def456"
}
```

---

## Tools

### GET /api/tools

List all registered tools (built-in and dynamic).

**Response:**
```json
[
  {
    "name": "get_user_orders",
    "description": "Get recent orders for a user",
    "parameters": [...]
  }
]
```

### POST /api/tool/register

Register a new dynamic tool.

**Request Body:**
```json
{
  "name": "get_user_orders",
  "description": "Get recent orders for a user",
  "datasource": "my-postgres",
  "database": "mydb",
  "sql": "SELECT * FROM orders WHERE user_id = :user_id LIMIT :limit",
  "parameters": [
    {"name": "user_id", "type": "integer", "required": true},
    {"name": "limit", "type": "integer", "default": 10}
  ]
}
```

### POST /api/tool/unregister

Remove a dynamic tool.

**Request Body:**
```json
{
  "tool_name": "get_user_orders"
}
```

### POST /api/tool/{name}

Execute a registered tool by name.

**Request Body:**
```json
{
  "user_id": 42,
  "limit": 5
}
```

---

## MCP Endpoint

### POST /mcp

JSON-RPC 2.0 endpoint for MCP protocol.

**Request Body:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "list_datasources",
    "arguments": {}
  }
}
```

See [MCP Tools](mcp-tools) for all available tool names and parameters.
