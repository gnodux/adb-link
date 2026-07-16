---
name: adb-link
description: Universal database gateway via MCP. Use when the user asks to query databases, explore schemas, list datasources, execute SQL, or manage database connections across MySQL, PostgreSQL, SQLite, ClickHouse, MSSQL, Elasticsearch, Hive, GaussDB, Oracle, TiDB, Redis, MongoDB, or Milvus.
---

# adb-link — Database Gateway for AI Agents

## Overview

adb-link exposes a unified MCP (Model Context Protocol) interface to 13+ database engines. Once configured, you can discover datasources, browse schemas, and execute queries through standard MCP tool calls — no SQL driver setup needed in the agent.

## Quick Start

If adb-link is not yet installed, run the interactive setup:

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/skills/adb-link/scripts/setup-mcp.sh | bash
```

The setup presents three modes:

| # | Mode | What it does |
|---|------|-------------|
| 1 | **CLI only** | Installs the binary for command-line use. No Agent configuration. |
| 2 | **Remote MCP** | Configures your Agent to connect to a remote adb-link HTTP endpoint. No binary install. |
| 3 | **Local MCP stdio** | Installs the binary + configures your Agent to use local MCP via stdio. (default) |

For non-interactive use, pass `--mode cli`, `--mode remote --url <url>`, or `--mode stdio`. See `references/platform-setup.md` for manual per-platform setup.

## MCP Transport Modes

| Mode | Command | Use case |
|------|---------|----------|
| stdio | `adb-link run-mcp` | Single-agent local use (default for Claude Desktop, Cursor) |
| HTTP | `adb-link run-all` | Multi-client / remote access, MCP at `http://host:8000/mcp` |

## Tool Overview

| Tool | Description |
|------|-------------|
| `list_datasources` | List all configured datasources with type, description, server info |
| `list_databases` | List databases in a datasource |
| `get_schema` | Get full schema (tables, columns, types, comments) for a database |
| `get_table_info` | Get detailed column info for a specific table |
| `get_view_info` | Get detailed column info for a specific view |
| `execute_query` | Execute SQL/DSL query, return structured results |
| `explain_query` | Get execution plan for a SQL statement |
| `submit_async_query` | Submit long-running query, returns query_id |
| `get_async_query_status` | Check async query status |
| `get_async_query_result` | Retrieve completed async query result |
| `register_datasource` | Dynamically add a new datasource (validated before persisting) |
| `unregister_datasource` | Remove a datasource and close connections |
| `register_tool` | Create a parameterized query tool (SQL template + JSON Schema) |
| `unregister_tool` | Remove a custom tool |

For full parameter details and return shapes, see `references/mcp-tools.md`.

## Typical Workflow

```
1. list_datasources → discover what's available
2. list_databases(datasource_name) → pick a database
3. get_schema(datasource_name, database) → understand table structure
4. get_table_info(datasource_name, database, table) → inspect columns
5. execute_query(datasource_name, database, sql) → run the query
```

### Example: Query a PostgreSQL database

```
list_datasources
  → [{"name":"my-pg","type":"postgresql","description":"Production DB"}]

list_databases(datasource_name="my-pg")
  → [{"name":"appdb","comment":"Main application"}]

get_schema(datasource_name="my-pg", database="appdb")
  → {"tables":[{"name":"users","columns":[{"name":"id","type":"INT4"},...]}]}

execute_query(datasource_name="my-pg", database="appdb", sql="SELECT count(*) FROM users", limit=10)
  → {"columns":["count"],"rows":[["42"]],"row_count":1}
```

## Configuration

Config directory: `~/.adb-link/conf/` (override with `ADB_LINK_CONFIG_DIR`)

Key config files:
- `mcp_stdio.yaml` — auth + permissions + datasources for stdio mode (copy from `mcp_stdio.yaml.example`)
- `datasource.yaml` — datasource definitions
- `auth.yaml` — API keys and users
- `permission.yaml` — RBAC rules

The stdio transport uses `mcp_stdio_user` as the default identity. Configure permissions for this user in your config.

YAML files support `${ENV_VAR}` interpolation for passwords and secrets.

## Supported Databases

MySQL, PostgreSQL, SQLite, ClickHouse, MSSQL, Elasticsearch, Hive, GaussDB, Oracle, TiDB, Redis, MongoDB, Milvus.

## Resources

- `references/mcp-tools.md` — Full tool parameter reference
- `references/platform-setup.md` — Per-platform MCP configuration (Claude Desktop, Cursor, Windsurf, Qoder CLI)
- `scripts/setup-mcp.sh` — Automated install + MCP registration script
