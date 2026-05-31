<div align="center">

# ADB-Link

**Bridging AI Agents to Your Databases**

A lightweight, high-performance database gateway designed for AI agents — providing unified SQL access, schema discovery, and tool orchestration across 13 database engines via REST API and MCP (Model Context Protocol).

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go)](https://go.dev)
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
| **13 Database Engines** | MySQL, PostgreSQL, SQLite, ClickHouse, MSSQL, Elasticsearch, Hive, GaussDB, Oracle, TiDB, Redis, MongoDB, Milvus |
| **MCP Protocol** | Full JSON-RPC 2.0 implementation (stdio + HTTP transport) |
| **Dynamic Datasource Registry** | Register/unregister datasources at runtime via API or MCP, with connection validation and config persistence |
| **Dynamic Tool Registry** | Register/unregister parameterized query tools at runtime via API or MCP |
| **Async Query Engine** | Submit long-running queries, poll status, retrieve results |
| **Schema Discovery** | Databases, tables, views, columns with type & comment info |
| **Hot Reload** | YAML config changes are detected and applied within seconds |
| **RBAC Permissions** | Glob-based access control on datasources, databases, tables, fields, and tools |
| **Connection Health** | Auto-ping, periodic health checks, fail-fast on unreachable hosts |
| **Pure Go** | Zero CGO dependencies — single static binary, cross-compile anywhere |

---

## Quick Start

### Prerequisites

- Go 1.25+ (for building from source)
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
kind: users
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

## Supported Databases

| Type | `type` value | SQL Dialect | Non-SQL Client |
|------|-------------|-------------|----------------|
| MySQL | `mysql` | MySQL | — |
| PostgreSQL | `postgresql` | PostgreSQL | — |
| SQLite | `sqlite` | SQLite | — |
| ClickHouse | `clickhouse` | ClickHouse | — |
| SQL Server | `mssql` | MSSQL | — |
| Elasticsearch | `elasticsearch` | JSON DSL | ESClient |
| Hive | `hive` | HiveQL | — |
| GaussDB | `gaussdb` | PostgreSQL-compat | — |
| Oracle | `oracle` | Oracle | — |
| TiDB | `tidb` | MySQL-compat | — |
| Redis | `redis` | Redis commands | RedisClient |
| MongoDB | `mongodb` | JSON filter/pipeline | MongoClient |
| Milvus | `milvus` | JSON query/search | MilvusClient |

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

#### Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Liveness check |

#### Datasources

| Method | Path                          | Description                       |
| --------| -------------------------------| -----------------------------------|
| POST   | `/api/datasources/list`       | List all configured datasources   |
| POST   | `/api/datasources/detail`     | Get datasource details            |
| POST   | `/api/datasources/test`       | Test datasource connectivity      |
| POST   | `/api/datasources/register`   | Dynamically register a datasource |
| POST   | `/api/datasources/unregister` | Unregister a datasource           |

#### Schema

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/databases/list` | List databases |
| POST | `/api/schema/get` | Get full schema (tables + views) |
| POST | `/api/schema/table` | Get table column info |
| POST | `/api/schema/view` | Get view column info |

#### Query

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/query/execute` | Execute SQL/query |
| POST | `/api/query/explain` | Get execution plan |

#### Async Query

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/async/query/submit` | Submit async query |
| POST | `/api/async/query/status` | Poll async query status |
| POST | `/api/async/query/result` | Retrieve async query result |
| POST | `/api/async/query/cancel` | Cancel async query |

#### Tools

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/tools` | List registered tools |
| POST | `/api/tool/register` | Register a new tool |
| POST | `/api/tool/unregister` | Unregister a tool |
| POST | `/api/tool/{name}` | Execute a tool |

#### Async Tool Execution

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/tool/async/{name}/submit` | Submit async tool execution |
| POST | `/api/tool/async/{name}/status` | Poll async tool status |
| POST | `/api/tool/async/{name}/result` | Retrieve async tool result |
| POST | `/api/tool/async/{name}/cancel` | Cancel async tool |

#### MCP

| Method | Path | Description |
|--------|------|-------------|
| POST | `/mcp` | MCP JSON-RPC endpoint (HTTP transport) |

### MCP Tools

All MCP tools are available via `tools/call`:

**Schema & Query**
- `list_datasources` — List all configured datasources
- `list_databases` — List databases in a datasource
- `get_schema` — Get full schema (tables, views, columns)
- `get_table_info` / `get_view_info` — Column details
- `execute_query` — Run SQL (or DSL for Elasticsearch, commands for Redis, filters for MongoDB/Milvus)
- `explain_query` — Get execution plan (MySQL, PostgreSQL, SQLite, ClickHouse, GaussDB, TiDB, MSSQL)

**Async Queries**
- `submit_async_query` — Submit a long-running query
- `submit_async_tool` — Submit a tool for async execution
- `get_async_query_status` / `get_async_query_result` — Poll and retrieve async results

**Dynamic Tool Management**
- `register_tool` / `unregister_tool` — Register/unregister parameterized query tools

**Dynamic Datasource Management**
- `register_datasource` / `unregister_datasource` — Register/unregister datasources with connection validation

**Dynamic Tools**
- Any tool registered via `register_tool` becomes immediately available as an MCP tool

### Configuration

All configuration is YAML-based in the `conf/` directory. Multiple documents can be combined in a single file using `---` separators.

| File | Kind | Purpose |
|------|------|---------|
| `datasource-*.yaml` | `datasource` | Database connection definitions |
| `auth.yaml` | `users` | API keys and user accounts |
| `permission-*.yaml` | `permission` | RBAC access control rules |
| `tool-*.yaml` | `tool` | Custom parameterized query tools |
| `toolset-*.yaml` | `toolset` | Tool grouping and organization |
| `metadata-*.yaml` | `metadata` | Column/table comments and annotations |

Environment variables are supported via `${VAR_NAME}` syntax. Configuration changes are hot-reloaded automatically via file watcher.

#### Datasource Config Example

```yaml
kind: datasource
name: my-mysql
type: mysql
description: "Production MySQL"
connection:
  host: ${DB_HOST}
  port: 3306
  username: root
  password: ${DB_PASSWORD}
  default_database: mydb
options:
  pool_size: 10
```

#### Auth Config Example

```yaml
kind: users
users:
  - name: admin
    api_key: "your-secret-key"
    group: admin
    email: "admin@example.com"
    description: "Administrator"
```

#### Permission Config Example

```yaml
kind: permission
users:
  - admin
rules:
  - datasource: "*"
    databases: ["*"]
    tables: ["*"]
    fields: ["*"]
```

#### Tool Config Example

```yaml
kind: tool
name: query_recent_users
description: "Query recently created users"
datasource: my-postgres
database: mydb
template: "SELECT * FROM users WHERE created_at > :since LIMIT :limit"
input_schema:
  type: object
  properties:
    since:
      type: string
      description: "Start date (YYYY-MM-DD)"
    limit:
      type: integer
      description: "Max rows"
      default: 10
  required: ["since"]
```

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

## Testing

```bash
# Unit tests + SQLite integration (no external dependencies)
make test

# Unit tests only
make test-unit

# SQLite integration tests only
make test-sqlite

# Full integration tests (requires podman)
make test-integration

# Generate coverage report
make test-coverage
```

Integration tests use podman to spin up real database containers (MySQL, PostgreSQL, ClickHouse, MSSQL, Elasticsearch, Redis, MongoDB, Milvus) and are gated behind the `integration` build tag.

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
