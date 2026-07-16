# adb-link — Agent Install Instructions

Copy and paste this entire document into any AI Agent conversation (Claude, Cursor, Windsurf, Qoder, etc.) to install adb-link and register it as an MCP server automatically.

---

## Instructions for the Agent

Please follow these steps to install adb-link and configure it as an MCP server:

### Step 1 — Install adb-link binary

Run the one-line installer:

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/scripts/install-adb-link.sh | bash
```

This installs the adb-link binary to `~/.adb-link/bin/adb-link` and creates a symlink at `~/.local/bin/adb-link`.

### Step 2 — Run the MCP setup script

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/skills/adb-link/scripts/setup-mcp.sh | bash
```

This script will:
1. Verify the binary is installed
2. Create a default config at `~/.adb-link/conf/mcp_stdio.yaml` (if it doesn't exist)
3. Detect your agent platform (Claude Desktop / Cursor / Windsurf / Qoder CLI) and register the MCP server automatically

### Step 3 — Add your databases

Edit `~/.adb-link/conf/mcp_stdio.yaml` and uncomment/add a datasource block. Example for PostgreSQL:

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

### Step 4 — Restart your Agent

Restart your Agent application (Claude Desktop / Cursor / Windsurf) or reload skills in Qoder CLI with `/skills reload`.

### Step 5 — Verify

Call the `list_datasources` MCP tool. You should see your configured datasources.

---

## Manual Platform Setup

If the automated setup doesn't detect your platform, see the platform-specific configs below.

### Claude Desktop

Config file:
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

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

### Cursor

Config file: `~/.cursor/mcp.json` (or `.cursor/mcp.json` for project-level)

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

### Qoder CLI

```bash
# Install skill
cp -r ~/.local/share/adb-link/skills/adb-link ~/.qoder/skills/ 2>/dev/null || \
  curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/skills/adb-link/SKILL.md \
    -o ~/.qoder/skills/adb-link/SKILL.md --create-dirs

# Register MCP server
qoder mcp add adb-link -- adb-link run-mcp
```

---

## Available MCP Tools

Once installed, adb-link exposes these tools:

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

---

## Troubleshooting

**`adb-link: command not found`** — Add `~/.local/bin` to your PATH:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

**No datasources returned** — Edit `~/.adb-link/conf/mcp_stdio.yaml` and add at least one datasource block.

**Permission denied** — Check `~/.adb-link/conf/mcp_stdio.yaml` has the `mcp_stdio_user` section with appropriate permissions.

More help: https://github.com/gnodux/adb-link
