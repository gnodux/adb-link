---
title: Getting Started
---

# Getting Started

## Installation

### One-liner Install

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/scripts/install-adb-link.sh | bash
```

This installs the latest release to `~/.adb-link/` and creates a symlink at `~/.local/bin/adb-link`.

### Build from Source

```bash
git clone https://github.com/gnodux/adb-link.git
cd adb-link
make build
# Binary: bin/adb-link
```

### Requirements

- Go 1.22+ (for building from source)
- No CGO dependencies -- pure Go, single static binary

---

## Initial Configuration

Configuration files live in `~/.adb-link/conf/` by default (override via `ADB_LINK_CONFIG_DIR`).

```bash
mkdir -p ~/.adb-link/conf
cp conf/mcp_stdio.yaml.example ~/.adb-link/conf/mcp_stdio.yaml
```

### Add a Datasource

Create `~/.adb-link/conf/datasource.yaml`:

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

### Add Authentication

Create `~/.adb-link/conf/auth.yaml`:

```yaml
kind: users
users:
  - name: admin
    api_key: "your-secret-api-key"
    group: admin
  - name: mcp_stdio_user
    group: mcp
    description: "Default user for MCP stdio transport"
```

### Set Permissions

Create `~/.adb-link/conf/permission.yaml`:

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

---

## Running

```bash
# API + MCP HTTP on a single port (default :8000)
adb-link run-all

# API only
adb-link run-api

# MCP over stdio (for IDE/agent integration)
adb-link run-mcp
```

### Verify

```bash
curl http://localhost:8000/api/health
# {"status":"ok"}
```

---

## MCP Integration

### Claude Desktop / Cursor

Add to your MCP client configuration (`claude_desktop_config.json` or equivalent):

```json
{
  "mcpServers": {
    "adb-link": {
      "command": "adb-link",
      "args": ["run-mcp"]
    }
  }
}
```

The stdio transport uses `mcp_stdio_user` as the default identity. Configure permissions for this user in your auth/permission YAML files.

### HTTP Transport

For remote or multi-client access:

```bash
adb-link run-all  # MCP available at /mcp endpoint
```

Clients connect to `http://host:8000/mcp` with Bearer token authentication.

---

## Next Steps

- [Configuration Reference](configuration) -- All config options
- [API Reference](api-reference) -- REST API endpoints
- [MCP Tools](mcp-tools) -- Available MCP tools
- [Database Support](databases) -- Supported databases and connection details
