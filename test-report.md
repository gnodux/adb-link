# ADB-Link Test Report

**Date**: 2026-05-30  
**Branch**: `20260530`  
**Commit**: `1150135`  
**Go Version**: 1.25+  

---

## Summary

| Metric | Value |
|--------|-------|
| **Total Tests** | 323 |
| **Passed** | 323 |
| **Failed** | 0 |
| **Skipped** | 0 |
| **Total Coverage** | **26.7%** |

---

## Per-Package Results

| Package | Tests | Result | Coverage |
|---------|-------|--------|----------|
| `internal/api` | 24 | PASS | 14.2% |
| `internal/apperr` | 17 | PASS | 100.0% |
| `internal/config` | 36 | PASS | 61.3% |
| `internal/dialects` | 47 | PASS | 24.1% |
| `internal/mcp` | 24 | PASS | 26.2% |
| `internal/models` | 8 | PASS | 100.0% |
| `internal/services` | 167 | PASS | 21.9% |
| `tests/integration` | — | PASS | n/a |

---

## Coverage Breakdown

### Fully Covered (100%)

- `internal/apperr` — All error types, constructors, wrapping, and status helpers.
- `internal/models` — All data model methods, `IsEnabled`, `BuildInputSchema`, context helpers.

### High Coverage (60%+)

- `internal/config` (61.3%) — Config loading, reload, env-var interpolation, tool registration/persistence, settings. Uncovered: file watcher (fsnotify goroutine), `AllToolsets`.

### Moderate Coverage (20–30%)

- `internal/dialects` (24.1%) — `BuildDSN` methods for all SQL dialects are fully tested. `GetDatabases`/`GetTableNames`/`GetTableInfo`/`GetViewInfo` require live database connections (covered by podman integration tests, not counted in unit coverage).
- `internal/mcp` (26.2%) — JSON-RPC server core (initialize, ping, tools/list, tools/call, register/unregister). HTTP transport and stdio are not unit-tested.
- `internal/services` (21.9%) — Permission service (100%), metadata service (100%), ES client (partial). Query service, connection service, async query service, and non-SQL clients require live backends.

### Low Coverage (<20%)

- `internal/api` (14.2%) — Middleware and utility functions are fully covered. HTTP handlers are not unit-tested (require service layer mocking or integration tests).

---

## Test Categories

### Unit Tests (in-package, no external deps)

- **API layer**: Bearer auth middleware, CORS, JSON encoding/decoding, user extraction.
- **Config**: YAML loading, multi-document support, env-var interpolation, reload, tool CRUD, settings defaults and overrides.
- **Dialects**: DSN construction for all 13 database types (MySQL, PostgreSQL, SQLite, ClickHouse, MSSQL, Elasticsearch, Hive, GaussDB, Oracle, TiDB, Redis, MongoDB, Milvus).
- **MCP**: JSON-RPC protocol handling, tool registration, notification callbacks.
- **Models**: Data structures, auth context, input schema building.
- **Services**: Permission RBAC (glob matching, filtering), metadata enhancement, ES client HTTP layer, query parameter conversion, async query state management.

### Integration Tests (SQLite — no podman needed)

- SQLite schema discovery, table/view introspection, query execution via the full service stack.

### Integration Tests (podman — requires `//go:build integration`)

- MySQL, PostgreSQL, ClickHouse, MSSQL, Elasticsearch, Hive container-based tests.
- Full lifecycle: container start → seed data → schema discovery → query execution → cleanup.

---

## Coverage Notes

The 26.7% unit coverage figure is expected for this project architecture:

1. **Database-dependent code** (dialects' query methods, connection service, query service, schema service) requires live database instances and is exercised by integration tests, not captured in unit coverage.
2. **HTTP handlers** in `internal/api` depend on the service layer and are validated through integration tests.
3. **File watcher** uses fsnotify goroutines that are difficult to unit-test deterministically.
4. **Non-SQL clients** (Redis, MongoDB, Milvus) require live servers and are covered by podman integration tests.

---

## Generated Artifacts

| File | Description |
|------|-------------|
| `coverage.out` | Go coverage profile (atomic mode) |
| `coverage.html` | Interactive HTML coverage report |
| `test-report.md` | This report |
