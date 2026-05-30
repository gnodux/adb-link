<div align="center">

# ADB-Link

**Bridging AI Agents to Your Databases**

A lightweight, high-performance database gateway designed for AI agents — providing unified SQL access, schema discovery, and tool orchestration across multiple database engines via REST API and MCP (Model Context Protocol).

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-2024--11--05-blueviolet?style=flat-square)](https://modelcontextprotocol.io)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey?style=flat-square)]()

</div>

---

## Demo

A complete workflow: discover datasources, explore schema, then execute queries — all via the MCP protocol.

```bash
# 1. List available datasources
curl -s -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-api-key>" \
  -d '{
    "jsonrpc": "2.0", "id": 1, "method": "tools/call",
    "params": {"name": "list_datasources", "arguments": {}}
  }'
# => [{"name":"my-postgres","type":"postgresql","description":"Production DB", ...}]

# 2. List databases in a datasource
curl -s -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-api-key>" \
  -d '{
    "jsonrpc": "2.0", "id": 2, "method": "tools/call",
    "params": {"name": "list_databases", "arguments": {"datasource_name": "my-postgres"}}
  }'
# => [{"name":"mydb","comment":"Main application database"}, ...]

# 3. Get schema (tables & columns)
curl -s -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-api-key>" \
  -d '{
    "jsonrpc": "2.0", "id": 3, "method": "tools/call",
    "params": {"name": "get_schema", "arguments": {"datasource_name": "my-postgres", "database": "mydb"}}
  }'
# => {"tables":[{"name":"users","columns":[{"name":"id","type":"INT4"},{"name":"username","type":"VARCHAR"}, ...]}]}

# 4. Get detailed table info
curl -s -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-api-key>" \
  -d '{
    "jsonrpc": "2.0", "id": 4, "method": "tools/call",
    "params": {"name": "get_table_info", "arguments": {"datasource_name": "my-postgres", "database": "mydb", "table": "users"}}
  }'
# => {"name":"users","columns":[{"name":"id","type":"INT4","nullable":false}, ...]}

# 5. Execute a query
curl -s -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-api-key>" \
  -d '{
    "jsonrpc": "2.0", "id": 5, "method": "tools/call",
    "params": {
      "name": "execute_query",
      "arguments": {
        "datasource_name": "my-postgres",
        "database": "mydb",
        "sql": "SELECT id, username, created_at FROM users WHERE created_at > '\''2026-01-01'\'' ORDER BY created_at DESC LIMIT 10"
      }
    }
  }'
# => {"columns":[{"name":"id","type":"INT4"},...],"rows":[[1,"alice","2026-03-15"], ...],"row_count":3}

# 6. Register a reusable query tool
curl -s -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-api-key>" \
  -d '{
    "jsonrpc": "2.0", "id": 6, "method": "tools/call",
    "params": {
      "name": "register_tool",
      "arguments": {
        "name": "query_recent_users",
        "description": "Query users created after a given date",
        "datasource": "my-postgres",
        "database": "mydb",
        "template": "SELECT id, username, created_at FROM users WHERE created_at > :since ORDER BY created_at DESC LIMIT :limit",
        "input_schema": {
          "type": "object",
          "properties": {
            "since": {"type": "string", "description": "Start date (YYYY-MM-DD)"},
            "limit": {"type": "integer", "description": "Max rows to return", "default": 10}
          },
          "required": ["since"]
        }
      }
    }
  }'
# => {"success":true, "name":"query_recent_users", "persisted_to":"conf/tool-query_recent_users.yaml"}

# 7. Call the registered tool
curl -s -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-api-key>" \
  -d '{
    "jsonrpc": "2.0", "id": 7, "method": "tools/call",
    "params": {"name": "query_recent_users", "arguments": {"since": "2026-01-01", "limit": 5}}
  }'
# => {"columns":[...],"rows":[...],"row_count":3}
```

---

## Core Features

| Feature | Description |
|---------|-------------|
| **Multi-Database Support** | MySQL, PostgreSQL, ClickHouse, SQLite, SQL Server, Hive |
| **MCP Protocol** | Full JSON-RPC 2.0 implementation (stdio + HTTP transport) |
| **Dynamic Tool Registry** | Register/unregister query tools at runtime via API or MCP |
| **Async Query Engine** | Submit long-running queries, poll status, retrieve results |
| **Schema Discovery** | Databases, tables, views, columns with type & comment info |
| **Hot Reload** | YAML config changes are detected and applied within seconds |
| **RBAC Permissions** | Glob-based access control on datasources, databases, tables, fields, and tools |
| **Connection Health** | Auto-ping, periodic health checks, fail-fast on unreachable hosts |
| **Pure Go** | Zero CGO dependencies — single static binary, cross-compile anywhere |

---

## Quick Start

### Prerequisites

- Go 1.22+ (for building from source)
- One or more supported databases accessible on the network

### Install

```bash
# Clone and build
git clone https://github.com/gnodux/adb-link.git
cd adb-link
make build

# Or install directly
go install github.com/gnodux/adb-link/cmd/adb-link@latest
```

### Configure

Create a `conf/` directory with your datasource definitions:

```yaml
# conf/datasource.yaml
kind: datasource
name: my-postgres
type: postgresql
description: "Production PostgreSQL"
connection:
  host: 127.0.0.1
  port: 5432
  username: app_user
  password: ${PG_PASSWORD}   # supports env var interpolation
  default_database: mydb
options:
  pool_size: 10
```

Add authentication (optional but recommended):

```yaml
# conf/auth.yaml
kind: auth
users:
  - name: admin
    api_key: "your-secret-api-key"
    group: admin
```

### Run

```bash
# API + MCP HTTP on a single port (default :8000)
./bin/adb-link run-all

# API only
./bin/adb-link run-api

# MCP over stdio (for IDE/agent integration)
./bin/adb-link run-mcp
```

### Verify

```bash
curl http://localhost:8000/api/health
# {"status":"ok"}
```

---

## Documentation

### Architecture

```
┌─────────────────────────────────────────────────────┐
│                   AI Agent / Client                  │
└──────────┬─────────────────────────┬────────────────┘
           │ REST API                │ MCP (JSON-RPC)
           ▼                         ▼
┌──────────────────────────────────────────────────────┐
│                    ADB-Link Server                    │
│  ┌──────────┐ ┌───────────┐ ┌────────────────────┐  │
│  │  Router  │ │    MCP    │ │   Config Service   │  │
│  │  + Auth  │ │  Server   │ │  (Hot-Reload/YAML) │  │
│  └─────┬────┘ └─────┬─────┘ └────────────────────┘  │
│        │             │                               │
│  ┌─────▼─────────────▼───────────────────────────┐   │
│  │          Service Layer                        │   │
│  │  Schema · Query · Async · Permission · Meta   │   │
│  └──────────────────┬────────────────────────────┘   │
│                     │                                │
│  ┌──────────────────▼────────────────────────────┐   │
│  │         Connection Service (Pool + Health)    │   │
│  └──────────────────┬────────────────────────────┘   │
│                     │                                │
│  ┌──────────────────▼────────────────────────────┐   │
│  │   Dialect Layer (DSN Builder per DB engine)   │   │
│  └───────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────┘
           │         │         │         │
     ┌─────┘    ┌────┘    ┌────┘    ┌────┘
     ▼          ▼         ▼         ▼
  MySQL    PostgreSQL  ClickHouse  SQLite ...
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Liveness check |
| POST | `/api/datasources/list` | List datasources |
| POST | `/api/datasources/detail` | Datasource details |
| POST | `/api/datasources/test` | Test connectivity |
| POST | `/api/databases/list` | List databases |
| POST | `/api/schema/get` | Full schema |
| POST | `/api/schema/table` | Table info |
| POST | `/api/schema/view` | View info |
| POST | `/api/query/execute` | Execute SQL |
| POST | `/api/query/explain` | Explain plan |
| POST | `/api/async/query/submit` | Async query submit |
| POST | `/api/async/query/status` | Async query status |
| POST | `/api/async/query/result` | Async query result |
| POST | `/api/async/query/cancel` | Cancel async query |
| GET | `/api/tools` | List registered tools |
| POST | `/api/tool/register` | Register tool |
| POST | `/api/tool/unregister` | Unregister tool |
| POST | `/api/tool/{name}` | Execute tool |
| POST | `/mcp` | MCP JSON-RPC endpoint |

### MCP Tools

All MCP tools are available via `tools/call`:

- `list_datasources` — List all datasources
- `list_databases` — List databases in a datasource
- `get_schema` — Get full schema
- `get_table_info` / `get_view_info` — Column details
- `execute_query` — Run SQL
- `explain_query` — Execution plan
- `submit_async_query` — Async query
- `get_async_query_status` / `get_async_query_result` — Poll async results
- `register_tool` / `unregister_tool` — Dynamic tool management
- *Any dynamically registered tool*

### Configuration

All configuration is YAML-based in the `conf/` directory:

| File | Kind | Purpose |
|------|------|---------|
| `datasource.yaml` | `datasource` | Database connection definitions |
| `auth.yaml` | `auth` | API keys and users |
| `permission-*.yaml` | `permission` | RBAC rules |
| `tool-*.yaml` | `tool` | Custom query tools |
| `metadata-*.yaml` | `metadata` | Column/table comments |

Environment variables are supported via `${VAR_NAME}` syntax. Configuration changes are hot-reloaded automatically.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADB_LINK_CONFIG_DIR` | `./conf` | Config directory path |
| `ADB_LINK_API_HOST` | `0.0.0.0` | API bind address |
| `ADB_LINK_API_PORT` | `8000` | API bind port |
| `ADB_LINK_LOG_LEVEL` | `info` | Log level |
| `ADB_LINK_LOG_DIR` | `./logs` | Log directory |
| `ADB_LINK_RELOAD` | `true` | Enable hot-reload |
| `ADB_LINK_ASYNC_QUERY_TTL` | `3600` | Async result TTL (seconds) |

---

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

```bash
# Development workflow
make fmt       # Format code
make vet       # Run go vet
make test      # Run tests
make build     # Build binary
```

---

<div align="center">

**If you find ADB-Link useful, please give it a star!**

</div>
