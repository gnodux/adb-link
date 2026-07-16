# Platform Setup Guide

How to configure adb-link MCP in each supported agent platform.

## Prerequisites

1. Install adb-link binary:
   ```bash
   curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/scripts/install-adb-link.sh | bash
   ```
2. Create default config:
   ```bash
   mkdir -p ~/.adb-link/conf
   cp ~/.adb-link/conf/mcp_stdio.yaml.example ~/.adb-link/conf/mcp_stdio.yaml 2>/dev/null || \
     curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/conf/mcp_stdio.yaml.example \
       -o ~/.adb-link/conf/mcp_stdio.yaml
   ```
3. Edit `~/.adb-link/conf/mcp_stdio.yaml` to add your datasources.

---

## Claude Desktop

Config file locations:
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

Add adb-link as a stdio MCP server:

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

For HTTP mode (if running `adb-link run-all` on a remote server):

```json
{
  "mcpServers": {
    "adb-link": {
      "command": "npx",
      "args": ["mcp-remote", "http://your-server:8000/mcp"],
      "env": {
        "MCP_AUTH_TOKEN": "your-api-key"
      }
    }
  }
}
```

Restart Claude Desktop after editing.

---

## Cursor

Config file locations:
- **Global**: `~/.cursor/mcp.json`
- **Project-level**: `.cursor/mcp.json` in your project root

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

---

## Windsurf

Config file location:
- **macOS/Linux**: `~/.codeium/windsurf/mcp_config.json`
- **Windows**: `%USERPROFILE%\.codeium\windsurf\mcp_config.json`

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

---

## Qoder CLI

Install the skill package to Qoder CLI's skills directory:

```bash
# User-level (available in all projects)
cp -r skills/adb-link ~/.qoder/skills/

# Or project-level
cp -r skills/adb-link .qoder/skills/
```

Register adb-link as an MCP server using Qoder CLI:

```bash
qoder mcp add adb-link -- adb-link run-mcp
```

Alternatively, add to your Qoder MCP config manually:

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

Reload skills with `/skills reload` and verify with `/skills list`.

---

## Verification

After configuration, verify adb-link is registered:

```bash
adb-link version
```

In your agent, try calling `list_datasources` — it should return your configured datasources (or an empty array if none configured yet).
