---
name: adb-link
description: >
  Query databases through adb-link — a universal database gateway supporting
  MySQL, PostgreSQL, SQLite, ClickHouse, MSSQL, Elasticsearch, Redis, MongoDB,
  Milvus, Hive, GaussDB, Oracle, and TiDB via MCP tools or REST API.
whenToUse: >
  Use when the user wants to query, inspect, or explore a database connected
  through adb-link. Covers schema discovery, SQL execution, async queries,
  dynamic tool registration, and datasource management.
tools:
  - list_datasources
  - list_databases
  - get_schema
  - get_table_info
  - list_views
  - get_view_info
  - execute_query
  - explain_query
  - submit_async_query
  - get_async_query_status
  - get_async_query_result
  - register_tool
  - unregister_tool
  - register_datasource
  - unregister_datasource
---

# adb-link Skill

adb-link is a universal database gateway that exposes MCP tools for querying multiple database types through a unified interface. This skill guides the agent on how to use those tools effectively.

## Prerequisites

1. **adb-link installed**: Run the one-line installer or download from [GitHub Releases](https://github.com/gnodux/adb-link/releases).
2. **MCP server running**: Either `adb-link run-mcp` (stdio) or `adb-link run-all` (HTTP).
3. **Datasources configured**: At least one datasource defined in `~/.adb-link/conf/`.

## Available MCP Tools

### Schema Discovery

| Tool | Description | Key Arguments |
|------|-------------|---------------|
| `list_datasources` | List all configured datasources | — |
| `list_databases` | List databases in a datasource | `datasource_name` |
| `get_schema` | Get full schema (tables + columns) | `datasource_name`, `database` |
| `get_table_info` | Detailed column info for a table | `datasource_name`, `database`, `table` |
| `list_views` | List views in a database | `datasource_name`, `database` |
| `get_view_info` | Detailed info for a view | `datasource_name`, `database`, `view` |

### Query Execution

| Tool | Description | Key Arguments |
|------|-------------|---------------|
| `execute_query` | Execute SQL/DSL query (synchronous) | `datasource_name`, `database`, `sql` |
| `explain_query` | Get execution plan for a query | `datasource_name`, `database`, `sql` |
| `submit_async_query` | Submit long-running query (async) | `datasource_name`, `database`, `sql` |
| `get_async_query_status` | Check async query status | `query_id` |
| `get_async_query_result` | Retrieve async query result | `query_id` |

### Dynamic Management

| Tool | Description | Key Arguments |
|------|-------------|---------------|
| `register_tool` | Register a parameterized SQL tool | `name`, `description`, `datasource`, `database`, `sql`, `parameters` |
| `unregister_tool` | Remove a dynamic tool | `tool_name` |
| `register_datasource` | Register a new datasource at runtime | `name`, `type`, `connection` |
| `unregister_datasource` | Remove a datasource | `datasource_name` |

## Typical Workflows

### Explore a database

```
1. list_datasources                                    → find available datasources
2. list_databases(datasource_name="pg")                → find databases
3. get_schema(datasource_name="pg", database="mydb")   → see all tables and columns
4. execute_query(datasource_name="pg", database="mydb", sql="SELECT * FROM users LIMIT 5")
```

### Run a long query asynchronously

```
1. submit_async_query(datasource_name="ch", database="analytics", sql="SELECT ...")
   → returns {"query_id": "abc123"}
2. get_async_query_status(query_id="abc123")
   → wait until status == "completed"
3. get_async_query_result(query_id="abc123")
   → retrieve rows
```

### Register a reusable parameterized tool

```
register_tool(
  name="get_orders_by_status",
  description="Get orders filtered by status",
  datasource="pg",
  database="shop",
  sql="SELECT * FROM orders WHERE status = ':status' LIMIT :limit",
  parameters=[
    {"name": "status", "type": "string", "required": true, "description": "Order status"},
    {"name": "limit",  "type": "integer", "required": false, "description": "Max rows"}
  ]
)
```

## Supported Database Types

`mysql` · `postgresql` · `sqlite` · `clickhouse` · `mssql` · `elasticsearch` · `redis` · `mongodb` · `milvus` · `hive` · `gaussdb` · `oracle` · `tidb`

## Permission Notes

- MCP stdio transport uses `mcp_stdio_user` as the default identity.
- Ensure that user has permissions configured in `~/.adb-link/conf/` for the target datasources.
- Empty/anonymous users bypass all permission checks (suitable for local development).

## Resources

- [Documentation](https://github.com/gnodux/adb-link/tree/main/docs)
- [MCP Tools Reference](https://github.com/gnodux/adb-link/blob/main/docs/mcp-tools.md)
- [Configuration Guide](https://github.com/gnodux/adb-link/blob/main/docs/configuration.md)
- [One-click Install Guide](../docs/install-mcp-agents.md)
