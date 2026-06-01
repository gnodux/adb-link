---
title: API 参考
---

# REST API 参考

所有 API 端点需要 Bearer Token 认证（`/api/health` 除外）。

```
Authorization: Bearer <your-api-key>
```

基础 URL: `http://localhost:8000`

---

## 健康检查

### GET /api/health

存活检查，无需认证。

**响应：**
```json
{"status": "ok"}
```

---

## 数据源

### POST /api/datasources/list

列出所有已配置的数据源。

**请求体：**
```json
{}
```

**响应：**
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

获取数据源详细信息。

**请求体：**
```json
{
  "datasource_name": "my-postgres"
}
```

### POST /api/datasources/test

测试数据源连通性。

**请求体：**
```json
{
  "datasource_name": "my-postgres"
}
```

**响应：**
```json
{
  "status": "ok",
  "message": "Connection successful"
}
```

### POST /api/datasources/register

运行时注册新数据源（无需重启）。

**请求体：**
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

运行时移除数据源。

**请求体：**
```json
{
  "datasource_name": "new-db"
}
```

---

## 数据库

### POST /api/databases/list

列出数据源中可用的数据库。

**请求体：**
```json
{
  "datasource_name": "my-postgres"
}
```

**响应：**
```json
[
  {"name": "mydb", "comment": "主应用数据库"},
  {"name": "analytics", "comment": ""}
]
```

---

## Schema

### POST /api/schema/get

获取数据库完整 Schema（表和列信息）。

**请求体：**
```json
{
  "datasource_name": "my-postgres",
  "database": "mydb"
}
```

**响应：**
```json
{
  "tables": [
    {
      "name": "users",
      "columns": [
        {"name": "id", "type": "INT4", "nullable": false, "comment": "主键"},
        {"name": "username", "type": "VARCHAR", "nullable": false, "comment": ""}
      ]
    }
  ]
}
```

### POST /api/schema/table

获取指定表的详细信息。

**请求体：**
```json
{
  "datasource_name": "my-postgres",
  "database": "mydb",
  "table": "users"
}
```

### POST /api/schema/view

获取指定视图的详细信息。

**请求体：**
```json
{
  "datasource_name": "my-postgres",
  "database": "mydb",
  "view": "active_users_view"
}
```

---

## 查询

### POST /api/query/execute

执行 SQL 查询。

**请求体：**
```json
{
  "datasource_name": "my-postgres",
  "database": "mydb",
  "sql": "SELECT id, username FROM users LIMIT 10"
}
```

**响应：**
```json
{
  "columns": ["id", "username"],
  "rows": [[1, "alice"], [2, "bob"]],
  "row_count": 2
}
```

### POST /api/query/explain

获取查询执行计划。

**请求体：**
```json
{
  "datasource_name": "my-postgres",
  "database": "mydb",
  "sql": "SELECT * FROM users WHERE id = 1"
}
```

---

## 异步查询

### POST /api/async/query/submit

提交长时间运行的异步查询。

**请求体：**
```json
{
  "datasource_name": "my-postgres",
  "database": "mydb",
  "sql": "SELECT * FROM large_table"
}
```

**响应：**
```json
{
  "query_id": "abc123-def456"
}
```

### POST /api/async/query/status

查询异步查询状态。

**请求体：**
```json
{
  "query_id": "abc123-def456"
}
```

**响应：**
```json
{
  "query_id": "abc123-def456",
  "status": "completed"
}
```

### POST /api/async/query/result

获取已完成的异步查询结果。

**请求体：**
```json
{
  "query_id": "abc123-def456"
}
```

### POST /api/async/query/cancel

取消正在运行的异步查询。

**请求体：**
```json
{
  "query_id": "abc123-def456"
}
```

---

## 工具

### GET /api/tools

列出所有已注册的工具（内置和动态）。

**响应：**
```json
[
  {
    "name": "get_user_orders",
    "description": "获取用户最近订单",
    "parameters": [...]
  }
]
```

### POST /api/tool/register

注册新的动态工具。

**请求体：**
```json
{
  "name": "get_user_orders",
  "description": "获取用户最近订单",
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

移除动态工具。

**请求体：**
```json
{
  "tool_name": "get_user_orders"
}
```

### POST /api/tool/{name}

按名称执行已注册的工具。

**请求体：**
```json
{
  "user_id": 42,
  "limit": 5
}
```

---

## MCP 端点

### POST /mcp

MCP 协议 JSON-RPC 2.0 端点。

**请求体：**
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

详见 [MCP 工具](mcp-tools) 了解所有可用工具名和参数。
