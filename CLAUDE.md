# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

adb-link is a lightweight database gateway written in Go that gives AI agents unified access to 13 database engines via REST API and MCP (Model Context Protocol). It is pure Go with zero CGO, producing a single static binary.

- **Module**: `github.com/gnodux/adb-link`
- **Go version**: 1.25.7 (uses Go 1.22+ stdlib `net/http` routing with `"METHOD /path"` patterns)
- **No web framework, no ORM, no external test libraries** — pure stdlib throughout

## Build & Run

```bash
make build              # Build binary → bin/adb-link
make run-all            # API + MCP HTTP on one port (default :8000)
make run-api            # REST API only
make run-mcp            # MCP stdio server
```

## Testing

```bash
make test               # Unit tests + SQLite integration (no external deps)
make test-unit          # Unit tests only (./internal/...)
make test-sqlite        # SQLite integration only
make test-integration   # Full podman-based integration (requires podman)
make test-coverage      # Generate coverage report
make lint               # go fmt + go vet
```

### Running a single test

```bash
go test ./internal/services/... -run TestConvertNamedParams -count=1 -v
go test ./tests/integration/... -run TestSQLite -count=1 -v
```

### Test structure

- **Unit tests**: in-package `*_test.go` files, table-driven, stdlib `testing` + `t.Helper()`. No external test frameworks.
- **SQLite integration** (`tests/integration/sqlite_test.go`): no build tag, runs with `go test ./...`. Creates temp SQLite DBs via `t.TempDir()`.
- **Podman integration** (`tests/integration/{postgres,mysql,...}_test.go`): build tag `//go:build integration`. `TestMain` checks `testutil.PodmanAvailable()` and skips if absent. Each test uses `testutil.StartContainer()` → `testutil.WaitForSQL()`/`WaitForHTTP()` → seed → test → auto-cleanup via `t.Cleanup()`.

## Architecture

```
cmd/adb-link/main.go    CLI entry: run-all, run-api, run-mcp, version subcommands
cmd/adb-link/drivers.go Blank imports for database/sql drivers

internal/
  api/        HTTP layer: Go 1.22+ ServeMux routing, BearerAuth + CORS middleware
  apperr/     Typed errors: apperr.Error with Code, HTTP Status, Msg, Cause
  config/     YAML config: env-var interpolation (${VAR}), atomic snapshot, fsnotify hot-reload
  dialects/   SchemaDialect interface — one impl per DB engine (BuildDSN, GetDatabases, etc.)
  mcp/        MCP JSON-RPC 2.0 server: HTTP transport (http.go) + stdio transport (stdio.go)
  models/     Shared data types only — no business logic
  services/   Business logic + DI container wiring all services

tests/
  integration/ Podman-based integration tests + SQLite tests
  testutil/    Container lifecycle, port allocation, readiness polling

conf/          Runtime YAML configs (datasource, auth, permission, tool, metadata)
```

### Request flow

```
Client → REST API (api/router.go → middleware → handlers)
       → MCP (mcp/http.go or mcp/stdio.go → JSON-RPC dispatch → tool handlers)
       → Services (services/*.go)
       → ConnectionService (cached *sql.DB or NonSQLClient)
       → SchemaDialect (SQL) or NonSQLClient (ES/Redis/Mongo/Milvus)
       → Database
```

### Key abstractions

- **`Container`** (`services/container.go`): DI container. `NewContainer()` wires all services; `Start()` launches health checks, async cleanup, and config hot-reload watcher; hot-reload callbacks refresh permissions/metadata and invalidate connections.
- **`ConfigService`** (`config/loader.go`): Loads multi-document YAML from `conf/` with `${ENV_VAR}` interpolation. Provides atomic snapshot access. Supports reload callbacks. Tool configs are persisted back to YAML files.
- **`SchemaDialect`** (`dialects/dialect.go`): Interface for SQL databases — `BuildDSN`, `GetDatabases`, `GetTableNames`, `GetViewNames`, `GetTableInfo`, `GetViewInfo`. One implementation per engine.
- **`NonSQLClient`** (`services/non_sql_client.go`): Interface for non-SQL databases (ES, Redis, MongoDB, Milvus) — `Ping`, `Close`, `GetDatabases`, `GetTableNames`, `GetTableInfo`, `Execute`. Uses native client libraries instead of `database/sql`.
- **`ConnectionService`** (`services/connection_service.go`): Cached connection pool for both SQL (`*sql.DB`) and NonSQL clients. Periodic health checks. `InvalidateAll()` called on config reload.
- **`PermissionService`** (`services/permission_service.go`): Glob-pattern RBAC on datasource/database/table/field/tool. Empty/anonymous user bypasses all checks.
- **`apperr.Error`** (`apperr/error.go`): Typed errors with stable error codes and HTTP status codes. Use these for all user-facing errors.

### Adding a new database type

1. Add a `DatabaseType` constant in `internal/models/datasource.go`
2. For SQL databases: implement `SchemaDialect` in `internal/dialects/<name>.go` and register in `GetDialect()` switch
3. For non-SQL databases: implement `NonSQLClient` in `internal/services/<name>_client.go`, add to `IsNonSQLType()`, and add connection logic in `ConnectionService`
4. Add blank driver import in `cmd/adb-link/drivers.go` (SQL only)
5. Add `DialectInfo` entry in `models.DialectInfoMap`
6. Write unit tests for the dialect and an integration test in `tests/integration/`

### SQL template tools

Tools defined in YAML (`conf/tools.yaml`, kind: `tool`) use `:param_name` placeholders. `convertNamedParams()` in `query_service.go` converts these to driver-specific placeholders: `?` for MySQL/SQLite, `$N` for PostgreSQL, `@pN` for MSSQL. Parameters are substituted from the tool's `input_schema` at execution time.

## Conventions

- Error handling: return `apperr.Error` for user-facing errors; wrap internal errors with context
- Permission checks: empty user string bypasses all permission filtering (used by stdio MCP transport)
- YAML configs use `kind:` discriminator for multi-document support (`---` separator)
- Config hot-reload: `ConfigService.AddReloadCallback()` triggers service state refresh and connection invalidation
- All logging uses `log/slog` (stdlib structured logging)
- Cross-compilation: `make build-linux`, `make build-darwin`, `make build-windows` (zero CGO enables this)
