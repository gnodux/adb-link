# Platform Setup Guide

How to configure adb-link MCP in each supported agent platform.

## Setup Modes

adb-link supports three setup modes. Use the interactive setup script or choose manually:

| Mode | Description | Prerequisites |
|------|-------------|---------------|
| **CLI only** | Command-line use, no Agent config | Install binary |
| **Remote MCP** | Connect Agent to remote server | No binary needed, just the server URL |
| **Local MCP stdio** | Agent uses local binary via stdio | Install binary |

```bash
# Interactive setup (presents mode selection menu)
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/skills/adb-link/scripts/setup-mcp.sh | bash

# Non-interactive with explicit mode
bash setup-mcp.sh --mode cli
bash setup-mcp.sh --mode remote --url http://your-server:8000/mcp
bash setup-mcp.sh --mode stdio
```

## Prerequisites (for CLI and Local MCP stdio modes)

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

**Remote MCP** (connect to a remote adb-link server without installing locally):

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
# Local stdio
qoder mcp add adb-link -- adb-link run-mcp

# Or remote MCP
qoder mcp add adb-link --url http://your-server:8000/mcp
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
