# adb-link — Agent Install Instructions

Copy and paste this entire document into any AI Agent conversation (Claude, Cursor, Windsurf, Qoder, etc.) to install adb-link and register it as an MCP server automatically.

---

## Instructions for the Agent

Please follow these steps to install adb-link and configure it as an MCP server:

### Step 1 — Ask me which setup mode I want

Present the following three options and wait for my answer:

1. **CLI only** — Install adb-link binary for command-line use. No Agent configuration.
2. **Remote MCP** — Connect my Agent to a remote adb-link MCP HTTP endpoint. No binary install needed.
3. **Local MCP stdio** (default) — Install adb-link binary and configure my Agent to use local MCP stdio.

### Step 2 — Execute based on my choice

**If I chose "CLI only" (mode 1):**

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/scripts/install-adb-link.sh | bash
```

Then verify with `adb-link version` and stop. No Agent configuration needed.

**If I chose "Remote MCP" (mode 2):**

Ask me for the remote MCP URL (e.g. `http://10.0.0.1:8000/mcp`), then configure my Agent's MCP config to add:

```json
{
  "mcpServers": {
    "adb-link": {
      "url": "<the URL I provided>"
    }
  }
}
```

Write this to the correct config file for my Agent platform (see "Manual Platform Setup" below). No binary install needed.

**If I chose "Local MCP stdio" (mode 3, default):**

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/skills/adb-link/scripts/setup-mcp.sh | bash
```

This script will:
1. Install the adb-link binary
2. Create a default config at `~/.adb-link/conf/mcp_stdio.yaml` (if it doesn't exist)
3. Detect my Agent platform and register the MCP server automatically

Or run with explicit flags for non-interactive use:

```bash
# --mode cli: CLI only
# --mode remote --url http://...: Remote MCP
# --mode stdio: Local MCP stdio (default)
bash setup-mcp.sh --mode stdio --agent claude
```

### Step 3 — Add your databases (modes 1 and 3 only)

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

### Cursor

Config file: `~/.cursor/mcp.json` (or `.cursor/mcp.json` for project-level)

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

### Windsurf

Config file: `~/.codeium/windsurf/mcp_config.json`

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

### Qoder CLI

```bash
# Install skill
cp -r ~/.local/share/adb-link/skills/adb-link ~/.qoder/skills/ 2>/dev/null || \
  curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/skills/adb-link/SKILL.md \
    -o ~/.qoder/skills/adb-link/SKILL.md --create-dirs

# Register MCP server (stdio)
qoder mcp add adb-link -- adb-link run-mcp

# Or register MCP server (remote)
qoder mcp add adb-link --url http://your-server:8000/mcp
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

**Remote MCP connection failed** — Ensure the remote adb-link server is running (`adb-link run-all`) and the URL is reachable.

More help: https://github.com/gnodux/adb-link
