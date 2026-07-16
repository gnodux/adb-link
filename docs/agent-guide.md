---
title: Agent Install Guide
---

# Agent Install Guide

Install adb-link and configure MCP for any AI Agent in minutes.

---

## Quick Setup

Run the one-click setup script — it auto-detects your Agent platform and handles everything:

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/skills/adb-link/scripts/setup-mcp.sh | bash
```

This script will:

1. Install the adb-link binary to `~/.adb-link/bin/`
2. Create a default config at `~/.adb-link/conf/mcp_stdio.yaml` (if it doesn't exist)
3. Detect your Agent platform (Claude Desktop / Cursor / Windsurf / Qoder CLI) and register the MCP server

After running, **restart your Agent** and call `list_datasources` to verify.

---

## Three Setup Modes

adb-link supports three installation modes. Choose the one that fits your workflow:

### Mode 1 — CLI Only

Install the binary for command-line use. No Agent configuration needed.

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/scripts/install-adb-link.sh | bash
```

Verify:

```bash
adb-link version
```

### Mode 2 — Remote MCP

Connect your Agent to a remote adb-link MCP HTTP endpoint. No binary install needed on your machine.

Ask your administrator for the MCP URL (e.g. `http://10.0.0.1:8000/mcp`), then add to your Agent's MCP config:

```json
{
  "mcpServers": {
    "adb-link": {
      "url": "http://your-server:8000/mcp"
    }
  }
}
```

### Mode 3 — Local MCP stdio (default)

Install adb-link binary and configure your Agent to use local MCP stdio transport.

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/skills/adb-link/scripts/setup-mcp.sh | bash
```

Or run with explicit flags:

```bash
# --mode cli: CLI only
# --mode remote --url http://...: Remote MCP
# --mode stdio: Local MCP stdio (default)
bash setup-mcp.sh --mode stdio --agent claude
```

---

## Paste-into-Agent Instructions

Copy the content of [AGENT_INSTALL.md](https://github.com/gnodux/adb-link/blob/main/AGENT_INSTALL.md) and paste it directly into any AI Agent conversation (Claude, Cursor, Windsurf, Qoder, etc.). The Agent will read the instructions and execute the setup automatically.

---

## Manual Platform Setup

If the automated setup doesn't detect your platform, configure your Agent manually.

### Claude Desktop

Config file location:

| OS | Path |
|----|------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |
| Linux | `~/.config/Claude/claude_desktop_config.json` |

**Local stdio:**

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

**Remote MCP:**

```json
{
  "mcpServers": {
    "adb-link": {
      "url": "http://your-server:8000/mcp"
    }
  }
}
```

Restart Claude Desktop after editing.

### Cursor

Config file: `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (project-level)

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

Cursor reads the config on startup. Reload the window to apply changes.

### Windsurf

Config file: `~/.codeium/windsurf/mcp_config.json`

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

Restart Windsurf after editing.

### Qoder CLI

Install the skill package:

```bash
# User-level (available in all projects)
cp -r skills/adb-link ~/.qoder/skills/
```

Register MCP server:

```bash
# stdio mode
qoder mcp add adb-link -- adb-link run-mcp

# or remote mode
qoder mcp add adb-link --url http://your-server:8000/mcp
```

Reload skills with `/skills reload` and verify with `/skills list`.

---

## Add Your Databases

After installation, edit `~/.adb-link/conf/mcp_stdio.yaml` to add datasource blocks:

```yaml
---
kind: datasource
name: my-postgres
type: postgresql
description: "My PostgreSQL database"
connection:
  host: 127.0.0.1
  port: 5432
  username: app_user
  password: ${PG_PASSWORD}
  default_database: mydb
```

Supported database types: `mysql`, `postgresql`, `sqlite`, `clickhouse`, `mssql`, `elasticsearch`, `hive`, `gaussdb`, `redis`, `mongodb`, `milvus`, `oracle`, `tidb`

See [Configuration](configuration) for full reference.

---

## Verify

Call the `list_datasources` MCP tool in your Agent. You should see your configured datasources.

```bash
# Or verify from CLI
adb-link version
```

---

## Troubleshooting

**`adb-link: command not found`** — Add `~/.local/bin` to your PATH:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

**No datasources returned** — Edit `~/.adb-link/conf/mcp_stdio.yaml` and add at least one datasource block.

**Permission denied** — Check `~/.adb-link/conf/mcp_stdio.yaml` has the `mcp_stdio_user` section with appropriate permissions.

**Remote MCP connection failed** — Ensure the remote adb-link server is running (`adb-link run-all`) and the URL is reachable.

More help: [GitHub Issues](https://github.com/gnodux/adb-link/issues)

---

## Available MCP Tools

Once installed, adb-link exposes these tools via MCP:

| Tool | Description |
|------|-------------|
| `list_datasources` | List all configured datasources |
| `list_databases` | List databases in a datasource |
| `get_schema` | Get full schema (tables + columns) |
| `get_table_info` | Get column details for a table |
| `get_view_info` | Get column details for a view |
| `execute_query` | Execute SQL/DSL queries |
| `explain_query` | Get SQL execution plan |
| `submit_async_query` | Submit long-running query |
| `register_datasource` | Add a new database connection |
| `register_tool` | Create parameterized query tools |

See [MCP Tools](mcp-tools) for full parameter reference.
