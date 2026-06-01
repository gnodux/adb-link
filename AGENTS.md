# AGENTS.md — adb-link

## Project Overview

adb-link is a universal database gateway that exposes MCP (Model Context Protocol) tools and a REST API for querying multiple database types through a unified interface.

## Architecture

```
cmd/adb-link/          — Entry point (run-all, run-api, run-mcp subcommands)
internal/
  api/                 — HTTP REST API + middleware (BearerAuth, CORS)
  apperr/              — Typed application errors with HTTP status mapping
  config/              — YAML config loader + settings (env-var interpolation)
  dialects/            — Database-specific schema introspection (SchemaDialect interface)
  mcp/                 — MCP JSON-RPC server + tool registration
  models/              — Shared data models (DatasourceConfig, QueryResult, etc.)
  services/            — Business logic (QueryService, SchemaService, ConnectionService, etc.)
tests/
  integration/         — Podman-based integration tests (build tag: integration)
  testutil/            — Test helpers (container lifecycle, port allocation, wait utilities)
conf/                  — Runtime YAML configuration (datasources, auth, permissions, tools)
```

## Supported Databases

| Type | SQL Dialect | Non-SQL Client |
|------|-------------|----------------|
| MySQL | MySQLDialect | — |
| PostgreSQL | PostgreSQLDialect | — |
| SQLite | SQLiteDialect | — |
| ClickHouse | ClickHouseDialect | — |
| MSSQL | MSSQLDialect | — |
| Elasticsearch | ElasticsearchDialect | ESClient |
| Hive | HiveDialect | — |
| GaussDB | GaussDBDialect | — |
| Oracle | OracleDialect | — |
| TiDB | TiDBDialect (MySQL-compat) | — |
| Redis | RedisDialect | RedisClient |
| MongoDB | MongoDBDialect | MongoClient |
| Milvus | MilvusDialect | MilvusClient |

## Build & Run

```bash
make build              # Build binary to bin/adb-link
make run-all            # API + MCP HTTP transport
make run-api            # REST API only
make run-mcp            # MCP stdio server
```

## Testing

```bash
make test               # Unit tests + SQLite integration (no podman needed)
make test-unit          # Unit tests only
make test-sqlite        # SQLite integration only
make test-integration   # Full podman integration (requires podman)
make test-coverage      # Generate coverage report
```

### Test Conventions

- **Unit tests**: In-package (`package services`), table-driven, stdlib `testing`
- **SQLite integration**: No build tag, runs with `go test ./...`
- **Podman integration**: `//go:build integration` tag, containers auto-cleaned via `t.Cleanup()`
- **No external test frameworks** — pure `testing` + `t.Helper()`

## Configuration

- Config directory: `conf/` (override via `ADB_LINK_CONFIG_DIR` env var)
- YAML files support env-var interpolation: `${ENV_VAR}`
- Hot-reload on file changes (fsnotify)
- Config kinds: `datasource`, `auth`, `permission`, `metadata`, `tool`

## Key Interfaces

- **SchemaDialect** (`internal/dialects/dialect.go`): BuildDSN, GetDatabases, GetTableNames, GetTableInfo, GetViewNames, GetViewInfo
- **NonSQLClient** (`internal/services/non_sql_client.go`): Ping, Close, GetDatabases, GetTableNames, GetTableInfo, Execute
- **MCP Server** (`internal/mcp/server.go`): JSON-RPC 2.0 with tools/list, tools/call

## Conventions

- Go 1.25+, no CGO for SQL drivers (modernc.org/sqlite)
- Error handling: use `internal/apperr` for typed errors with HTTP status codes
- Permission checks: empty/anonymous user bypasses all checks
- All YAML config files use `kind:` discriminator for multi-document support
- Connection pooling managed by ConnectionService with explicit invalidate/dispose

## Release

- **Version source**: `cmd/adb-link/main.go` 中的 `const version` 是版本唯一来源
- **Auto-increment**: 发版时如果用户未指定新版本号，根据当前版本自动自增 patch 版本号（例如 `0.1.0` → `0.1.1`）
- **Release workflow**: 每次发版必须按以下顺序执行：
  1. 更新 `cmd/adb-link/main.go` 中的 `const version` 值
  2. `git commit` 包含该文件变更
  3. `git tag v<version>` 打标签
  4. `git push` 推送代码和标签到远程
- **Tag naming**: tag 名格式为 `v<version>`，与 `main.go` 中的 version 保持一致
