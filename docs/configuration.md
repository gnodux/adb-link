---
title: Configuration
---

# Configuration

All configuration is YAML-based and lives in the config directory (`~/.adb-link/conf/` by default).

## Config Directory

| Variable | Default | Description |
|----------|---------|-------------|
| `ADB_LINK_CONFIG_DIR` | `~/.adb-link/conf` | Config directory path |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADB_LINK_API_HOST` | `0.0.0.0` | API bind address |
| `ADB_LINK_API_PORT` | `8000` | API bind port |
| `ADB_LINK_LOG_LEVEL` | `INFO` | Log level (DEBUG, INFO, WARN, ERROR) |
| `ADB_LINK_LOG_DIR` | `~/.adb-link/logs` | Log directory |
| `ADB_LINK_RELOAD` | `true` | Enable hot-reload |
| `ADB_LINK_ASYNC_QUERY_TTL` | `3600` | Async result TTL in seconds |

## Config File Types

Each YAML file uses a `kind:` discriminator:

| Kind | Purpose | Example File |
|------|---------|--------------|
| `datasource` | Database connection definitions | `datasource.yaml` |
| `users` | API keys and user accounts | `auth.yaml` |
| `permission` | RBAC access control rules | `permission.yaml` |
| `tool` | Custom query tool definitions | `tool-reports.yaml` |
| `metadata` | Column/table comment annotations | `metadata-mydb.yaml` |

---

## Datasource Configuration

```yaml
kind: datasource
name: my-postgres
type: postgresql
description: "Production PostgreSQL"
connection:
  host: 127.0.0.1
  port: 5432
  username: app_user
  password: ${PG_PASSWORD}
  default_database: mydb
options:
  pool_size: 10
```

### Supported Types

`mysql`, `postgresql`, `clickhouse`, `sqlite`, `mssql`, `hive`, `oracle`, `gaussdb`, `tidb`, `redis`, `mongodb`, `milvus`, `elasticsearch`

### Connection Fields

| Field | Description |
|-------|-------------|
| `host` | Database host |
| `port` | Database port |
| `username` | Connection username |
| `password` | Connection password (supports `${ENV_VAR}`) |
| `default_database` | Default database to connect to |
| `dsn` | Full DSN string (alternative to individual fields) |

### Options

| Option | Description |
|--------|-------------|
| `pool_size` | Connection pool size |
| `max_idle` | Maximum idle connections |
| `max_lifetime` | Maximum connection lifetime |

---

## Authentication (Users)

```yaml
kind: users
users:
  - name: admin
    api_key: "your-secret-api-key"
    group: admin
    email: "admin@example.com"
    description: "Administrator"
  - name: readonly
    api_key: "readonly-key"
    group: viewer
  - name: mcp_stdio_user
    group: mcp
    description: "Default user for MCP stdio transport"
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Username |
| `api_key` | No | Bearer token for API auth (supports `${ENV_VAR}`) |
| `group` | Yes | Permission group membership |
| `email` | No | User email |
| `description` | No | User description |

---

## Permission Rules

```yaml
kind: permission
groups: ["admin"]
enable: true
rules:
  - datasource: "*"
    databases: ["*"]
    tables: ["*"]
    fields: ["*"]
tools: ["*"]
```

### Fields

| Field | Description |
|-------|-------------|
| `groups` | List of groups this permission applies to |
| `enable` | Whether this permission set is active |
| `rules` | List of access rules |
| `rules[].datasource` | Datasource glob pattern |
| `rules[].databases` | Database glob patterns |
| `rules[].tables` | Table glob patterns |
| `rules[].fields` | Field glob patterns |
| `tools` | Tool glob patterns |

Glob patterns support `*` (match any) and exact names.

---

## Custom Tools

```yaml
kind: tool
name: get_user_orders
description: "Get recent orders for a user"
datasource: my-postgres
database: mydb
sql: "SELECT * FROM orders WHERE user_id = :user_id ORDER BY created_at DESC LIMIT :limit"
parameters:
  - name: user_id
    type: integer
    description: "User ID"
    required: true
  - name: limit
    type: integer
    description: "Max results"
    default: 10
```

### Tool Fields

| Field | Description |
|-------|-------------|
| `name` | Tool name (used in MCP `tools/call`) |
| `description` | Tool description shown to agents |
| `datasource` | Target datasource |
| `database` | Target database |
| `sql` | SQL template with named parameters (`:param`) |
| `parameters` | Parameter definitions with JSON Schema types |

---

## Metadata Annotations

```yaml
kind: metadata
datasource: my-postgres
database: mydb
tables:
  - name: users
    comment: "Application users"
    columns:
      - name: id
        comment: "Primary key"
      - name: status
        comment: "0=inactive, 1=active, 2=banned"
```

Metadata annotations enrich schema discovery with human-readable comments.

---

## Hot Reload

Configuration changes are automatically detected via filesystem notifications (fsnotify). Changes take effect within seconds without restarting the server.

Supported hot-reload operations:
- Add/remove/modify datasources
- Update users and permissions
- Add/remove custom tools
- Update metadata annotations

---

## Environment Variable Interpolation

All YAML config values support `${ENV_VAR}` syntax:

```yaml
connection:
  password: ${DB_PASSWORD}
  host: ${DB_HOST}
```

If the environment variable is not set, the literal `${VAR}` string is preserved.

---

## MCP Stdio Combined Config

For MCP stdio mode, a single file can contain all config kinds:

```yaml
kind: users
users:
  - name: mcp_stdio_user
    group: admin
---
kind: permission
groups: ["admin"]
enable: true
rules:
  - datasource: "*"
    databases: ["*"]
    tables: ["*"]
    fields: ["*"]
tools: ["*"]
---
kind: datasource
name: local-sqlite
type: sqlite
connection:
  dsn: "/path/to/database.db"
```

Multi-document YAML (separated by `---`) is supported.
