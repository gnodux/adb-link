---
title: Install adb-link in AI Agents
---

# Install adb-link in AI Agents

This guide shows how to connect adb-link as an MCP server in popular AI agents. Once connected, the agent gains direct access to all your configured databases through natural language queries.

## Prerequisites

**Install adb-link first** — before configuring any agent, you need the `adb-link` binary on your system.

### One-line install (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/scripts/install-adb-link.sh | bash
```

This script:
1. Detects your OS and architecture
2. Downloads the latest release from GitHub
3. Installs the binary to `~/.local/bin/adb-link`
4. Copies example config files to `~/.adb-link/conf/`

### Windows

Download the latest `.zip` from [GitHub Releases](https://github.com/gnodux/adb-link/releases), extract it, and add the folder to your `PATH`.

### Verify installation

```bash
adb-link version
```

---

## Configure datasources

After installing, configure at least one datasource before connecting to an agent:

```yaml
# ~/.adb-link/conf/datasource.yaml
kind: datasource
name: my-postgres
type: postgresql
host: localhost
port: 5432
user: postgres
password: ${DB_PASSWORD}
database: mydb
```

See the [Configuration Guide](configuration.md) for all supported database types and options.

---

## Agent Integration

### Claude Code

```bash
claude mcp add adb-link -- adb-link run-mcp
```

Then verify the server is recognized:

```bash
claude mcp list
```

### Claude Desktop / Cursor / Windsurf / Continue

Add the following to your agent's MCP configuration file:

**macOS / Linux:**

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

**Windows** (use the full path to avoid PATH resolution issues):

```json
{
  "mcpServers": {
    "adb-link": {
      "command": "C:\\Users\\YourName\\.local\\bin\\adb-link.cmd",
      "args": ["run-mcp"]
    }
  }
}
```

> **Config file locations:**
> - Claude Desktop (macOS): `~/Library/Application Support/Claude/claude_desktop_config.json`
> - Claude Desktop (Windows): `%APPDATA%\Claude\claude_desktop_config.json`
> - Cursor: `.cursor/mcp.json` in your project or `~/.cursor/mcp.json` globally
> - Windsurf: `~/.codeium/windsurf/mcp_config.json`

Restart the agent application after saving the config. The adb-link tools will appear in the agent's tool list.

### HTTP Transport (remote / multi-client)

If you prefer HTTP over stdio, or need to share one adb-link instance across multiple agents:

```bash
adb-link run-all   # Starts API + MCP at http://localhost:8000/mcp
```

Configure your agent to use the SSE endpoint:

```json
{
  "mcpServers": {
    "adb-link": {
      "url": "http://localhost:8000/mcp"
    }
  }
}
```

---

## One-Click Install Prompt

Copy the block below and paste it directly into any AI agent that has access to your filesystem and terminal. The agent will install adb-link and configure itself automatically.

---

```
Please help me install and configure adb-link as an MCP server on this machine.

Step 1 — Install the binary:
Run the following command in a terminal:
  curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/scripts/install-adb-link.sh | bash

If curl is not available or you are on Windows, download the latest release from:
  https://github.com/gnodux/adb-link/releases
and extract the binary to a directory in PATH.

Step 2 — Verify installation:
  adb-link version

Step 3 — Ask me which AI agent I am running in:
  (a) Claude Code  (b) Claude Desktop  (c) Cursor  (d) Windsurf  (e) Continue  (f) Other

Step 4 — Based on my answer, write the appropriate MCP configuration:
  • For Claude Code: run `claude mcp add adb-link -- adb-link run-mcp`
  • For others: add the JSON snippet from https://github.com/gnodux/adb-link/blob/main/docs/install-mcp-agents.md
    to the correct config file for my platform.

Step 5 — Confirm success by listing available MCP tools. The list should include
  list_datasources, execute_query, get_schema, and at least 10 other tools.
```

---

## Skill File (Qoder CLI)

If you use [Qoder CLI](https://qoder.ai), a ready-made Skill is included in this repository at [`skills/adb-link.md`](../skills/adb-link.md).

The Skill provides:
- Contextual guidance for using all adb-link MCP tools
- Workflow examples for schema exploration, query execution, and async queries
- Parameter references for every tool

The same Skill definition can be adapted for other agent platforms that support custom skill files.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| `command not found: adb-link` | Binary not in PATH | Add `~/.local/bin` to `PATH`, or use the full path in the MCP config |
| Agent shows 0 tools | adb-link failed to start | Run `adb-link run-mcp` in a terminal and check for errors |
| `permission denied` | No datasource permissions | Check `~/.adb-link/conf/` and ensure `mcp_stdio_user` has access |
| Windows: MCP fails to start | Path with spaces or missing `.cmd` shim | Use the `.cmd` shim path in the JSON config (see Windows example above) |

For more help: [GitHub Issues](https://github.com/gnodux/adb-link/issues)
