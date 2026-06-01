---
title: MCP 工具
---

# MCP 工具参考

ADB-Link 实现了 [Model Context Protocol (MCP)](https://modelcontextprotocol.io)，完整支持 JSON-RPC 2.0。所有工具通过 `tools/call` 方法调用。

## 传输模式

| 模式 | 命令 | 使用场景 |
|------|------|----------|
| stdio | `adb-link run-mcp` | IDE/智能体集成（Claude Desktop、Cursor） |
| HTTP | `adb-link run-all` | 远程访问、多客户端 |

---

## 工具列表

### list_datasources

列出所有可用数据源。

**参数：** 无

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

---

### list_databases

列出数据源中的数据库。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `datasource_name` | string | 是 | 目标数据源 |

---

### get_schema

获取数据库完整 Schema（表和列信息）。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `datasource_name` | string | 是 | 目标数据源 |
| `database` | string | 是 | 目标数据库 |

---

### get_table_info

获取指定表的详细列信息。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `datasource_name` | string | 是 | 目标数据源 |
| `database` | string | 是 | 目标数据库 |
| `table` | string | 是 | 表名 |

---

### get_view_info

获取指定视图的详细列信息。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `datasource_name` | string | 是 | 目标数据源 |
| `database` | string | 是 | 目标数据库 |
| `view` | string | 是 | 视图名 |

---

### execute_query

执行 SQL 或 DSL 查询。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `datasource_name` | string | 是 | 目标数据源 |
| `database` | string | 是 | 目标数据库 |
| `sql` | string | 是 | SQL/DSL 查询语句 |

**响应：**
```json
{
  "columns": ["id", "username", "created_at"],
  "rows": [[1, "alice", "2026-03-15"]],
  "row_count": 1
}
```

---

### explain_query

获取查询执行计划。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `datasource_name` | string | 是 | 目标数据源 |
| `database` | string | 是 | 目标数据库 |
| `sql` | string | 是 | 待分析的 SQL 查询 |

---

### submit_async_query

提交长时间运行的异步查询。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `datasource_name` | string | 是 | 目标数据源 |
| `database` | string | 是 | 目标数据库 |
| `sql` | string | 是 | SQL 查询 |

**响应：**
```json
{
  "query_id": "abc123-def456"
}
```

---

### get_async_query_status

查询异步查询状态。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `query_id` | string | 是 | 异步查询 ID |

状态值：`pending`、`running`、`completed`、`failed`、`cancelled`

---

### get_async_query_result

获取已完成的异步查询结果。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `query_id` | string | 是 | 异步查询 ID |

---

### register_tool

运行时注册动态查询工具。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 工具名称 |
| `description` | string | 是 | 工具描述 |
| `datasource` | string | 是 | 目标数据源 |
| `database` | string | 是 | 目标数据库 |
| `sql` | string | 是 | SQL 模板（使用 `:param` 占位符） |
| `parameters` | array | 是 | 参数定义 |

---

### unregister_tool

移除动态工具。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `tool_name` | string | 是 | 要移除的工具名 |

---

### register_datasource

运行时注册新数据源（带连接验证）。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 数据源名称 |
| `type` | string | 是 | 数据库类型 |
| `connection` | object | 是 | 连接详情 |

---

### unregister_datasource

运行时移除数据源。

**参数：**

| 名称 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `datasource_name` | string | 是 | 要移除的数据源名 |

---

## 动态工具

通过 `register_tool`（API 或 MCP）注册的工具立即可用。动态工具出现在 `tools/list` 中，调用方式与内置工具一致。

示例 -- 注册并调用自定义工具：

```json
// 1. 注册
{
  "jsonrpc": "2.0", "id": 1, "method": "tools/call",
  "params": {
    "name": "register_tool",
    "arguments": {
      "name": "get_active_users",
      "description": "获取最近 N 天活跃用户",
      "datasource": "my-postgres",
      "database": "mydb",
      "sql": "SELECT * FROM users WHERE last_active > NOW() - INTERVAL ':days days'",
      "parameters": [
        {"name": "days", "type": "integer", "required": true, "description": "天数"}
      ]
    }
  }
}

// 2. 调用新工具
{
  "jsonrpc": "2.0", "id": 2, "method": "tools/call",
  "params": {
    "name": "get_active_users",
    "arguments": {"days": 7}
  }
}
```

---

## 权限控制

MCP 工具遵循与 REST API 相同的 RBAC 权限体系。MCP stdio 传输使用 `mcp_stdio_user` 作为默认身份，请确保该用户配置了适当的权限。

空用户/匿名用户会跳过所有权限检查（适用于本地开发）。
