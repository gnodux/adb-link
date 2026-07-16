#!/usr/bin/env bash
#
# setup-mcp.sh — Install adb-link and configure MCP for your Agent.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/skills/adb-link/scripts/setup-mcp.sh | bash
#   bash setup-mcp.sh [--mode cli|remote|stdio] [--agent claude|cursor|windsurf|qoder|all] [--url <url>] [--dry-run]
#
# Modes:
#   cli     Install adb-link binary only (command-line usage, no Agent config)
#   remote  Connect your Agent to a remote adb-link MCP HTTP endpoint (no binary install)
#   stdio   Install binary + configure Agent to use local MCP stdio (default)
#
# Flags:
#   --mode    Setup mode: cli, remote, or stdio (default: interactive, or stdio in pipe)
#   --agent   Target agent platform (default: auto-detect, or "all")
#   --url     Remote MCP URL for --mode remote (e.g. http://10.0.0.1:8000/mcp)
#   --dry-run Show what would be done without making changes
#   --help    Show this help message

set -euo pipefail

REPO="gnodux/adb-link"
INSTALL_ROOT="${HOME}/.adb-link"
CONF_DIR="${INSTALL_ROOT}/conf"
LINK_DIR="${HOME}/.local/bin"
BIN_NAME="adb-link"

SETUP_MODE=""
AGENT_TARGET=""
REMOTE_URL=""
DRY_RUN=false

log()  { printf '\033[1;34m[adb-link]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[adb-link]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[adb-link]\033[0m %s\n' "$*" >&2; }
die()  { err "$*"; exit 1; }

# ---------- Parse args ----------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)   SETUP_MODE="$2"; shift 2 ;;
    --agent)  AGENT_TARGET="$2"; shift 2 ;;
    --url)    REMOTE_URL="$2"; shift 2 ;;
    --dry-run) DRY_RUN=true; shift ;;
    --help|-h)
      grep '^#' "$0" | sed 's/^# \?//'
      exit 0
      ;;
    *) die "Unknown argument: $1 (try --help)" ;;
  esac
done

# ---------- Dependency check ----------
require() {
  command -v "$1" >/dev/null 2>&1 || die "Missing required command: $1. Please install it and retry."
}

require curl
require tar
require uname

if [[ "$DRY_RUN" == true ]]; then
  log "DRY RUN mode — no changes will be made."
fi

# ---------- Mode selection ----------
is_pipe() {
  [[ ! -t 0 ]]
}

select_mode() {
  if [[ -n "$SETUP_MODE" ]]; then
    case "$SETUP_MODE" in
      cli|remote|stdio) ;;
      *) die "Invalid mode: $SETUP_MODE (use cli, remote, or stdio)" ;;
    esac
    return
  fi

  if is_pipe; then
    SETUP_MODE="stdio"
    log "Non-interactive mode detected — defaulting to: stdio"
    return
  fi

  echo ""
  printf '\033[1;36m'
  echo "╔═══════════════════════════════════════════════╗"
  echo "║         adb-link Setup — Choose Mode          ║"
  echo "╠═══════════════════════════════════════════════╣"
  echo "║                                               ║"
  echo "║  1) CLI only                                  ║"
  echo "║     Install adb-link binary for command-line  ║"
  echo "║     use. No Agent configuration.              ║"
  echo "║                                               ║"
  echo "║  2) Remote MCP                                ║"
  echo "║     Connect your Agent to a remote adb-link   ║"
  echo "║     MCP HTTP endpoint. No binary install.     ║"
  echo "║                                               ║"
  echo "║  3) Local MCP stdio              (default)    ║"
  echo "║     Install binary + configure your Agent to  ║"
  echo "║     use local adb-link via stdio protocol.    ║"
  echo "║                                               ║"
  echo "╚═══════════════════════════════════════════════╝"
  printf '\033[0m'
  echo ""

  local choice
  read -r -p "Enter choice [1/2/3] (default: 3): " choice
  case "${choice:-3}" in
    1) SETUP_MODE="cli" ;;
    2) SETUP_MODE="remote" ;;
    3) SETUP_MODE="stdio" ;;
    *) die "Invalid choice: $choice" ;;
  esac
}

# ---------- Remote URL input ----------
ask_remote_url() {
  if [[ -n "$REMOTE_URL" ]]; then
    if [[ "$REMOTE_URL" =~ ^https?:// ]]; then
      return
    fi
    die "Invalid URL: $REMOTE_URL (must start with http:// or https://)"
  fi

  if is_pipe; then
    die "Remote mode requires --url flag in non-interactive mode."
  fi

  local attempts=0
  while [[ $attempts -lt 3 ]]; do
    read -r -p "Enter remote MCP URL (e.g. http://10.0.0.1:8000/mcp): " REMOTE_URL
    if [[ "$REMOTE_URL" =~ ^https?:// ]]; then
      return
    fi
    attempts=$((attempts + 1))
    if [[ $attempts -lt 3 ]]; then
      warn "URL must start with http:// or https://. Try again ($attempts/3)."
    fi
  done
  die "Invalid URL after 3 attempts. Aborting."
}

# ---------- Step 1: Install binary ----------
install_binary() {
  log "Installing adb-link binary..."

  if command -v "$BIN_NAME" >/dev/null 2>&1; then
    local current_ver
    current_ver="$("$BIN_NAME" version 2>/dev/null || echo "unknown")"
    log "  adb-link already installed ($current_ver). Will update to latest."
  fi

  if [[ "$DRY_RUN" == true ]]; then
    log "  [dry-run] Would run: curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/install-adb-link.sh | bash"
    return
  fi

  curl -fsSL "https://raw.githubusercontent.com/${REPO}/main/scripts/install-adb-link.sh" | bash
}

# ---------- Step 2: Create default config ----------
create_config() {
  log "Setting up configuration..."

  local conf_file="${CONF_DIR}/mcp_stdio.yaml"

  if [[ "$DRY_RUN" == true ]]; then
    log "  [dry-run] Would ensure config at: ${conf_file}"
    return
  fi

  mkdir -p "$CONF_DIR"

  if [[ -f "$conf_file" ]]; then
    log "  Config already exists at ${conf_file} — preserving existing configuration."
  else
    log "  Creating default config from mcp_stdio.yaml.example..."
    curl -fsSL "https://raw.githubusercontent.com/${REPO}/main/conf/mcp_stdio.yaml.example" \
      -o "$conf_file"
    log "  Config created: ${conf_file}"
    warn "  Edit this file to add your database datasources."
  fi
}

# ---------- Step 3: Detect agent platform ----------
detect_agent() {
  if [[ -n "$AGENT_TARGET" ]]; then
    echo "$AGENT_TARGET"
    return
  fi

  local os
  os="$(uname -s)"

  local claude_config=""
  case "$os" in
    Darwin) claude_config="${HOME}/Library/Application Support/Claude/claude_desktop_config.json" ;;
    Linux)  claude_config="${HOME}/.config/Claude/claude_desktop_config.json" ;;
    MINGW*|MSYS*|CYGWIN*|Windows_NT) claude_config="${APPDATA}/Claude/claude_desktop_config.json" ;;
  esac

  if [[ -n "$claude_config" && -f "$claude_config" ]]; then
    echo "claude"
    return
  fi

  if [[ -f "${HOME}/.cursor/mcp.json" ]] || [[ -f ".cursor/mcp.json" ]]; then
    echo "cursor"
    return
  fi

  local windsurf_config="${HOME}/.codeium/windsurf/mcp_config.json"
  if [[ -f "$windsurf_config" ]]; then
    echo "windsurf"
    return
  fi

  if [[ -d "${HOME}/.qoder" ]]; then
    echo "qoder"
    return
  fi

  echo "all"
}

# ---------- Step 4: Register MCP ----------

agent_config_path() {
  local agent="$1"
  local os
  os="$(uname -s)"

  case "$agent" in
    claude)
      case "$os" in
        Darwin) echo "${HOME}/Library/Application Support/Claude/claude_desktop_config.json" ;;
        Linux)  echo "${HOME}/.config/Claude/claude_desktop_config.json" ;;
        MINGW*|MSYS*|CYGWIN*|Windows_NT) echo "${APPDATA}/Claude/claude_desktop_config.json" ;;
        *) die "Unsupported OS for Claude Desktop: $os" ;;
      esac
      ;;
    cursor) echo "${HOME}/.cursor/mcp.json" ;;
    windsurf) echo "${HOME}/.codeium/windsurf/mcp_config.json" ;;
  esac
}

merge_mcp_config() {
  local config_file="$1"
  local entry="$2"

  if [[ "$DRY_RUN" == true ]]; then
    log "  [dry-run] Would merge adb-link MCP entry into: ${config_file}"
    log "  [dry-run] Entry: ${entry}"
    return
  fi

  mkdir -p "$(dirname "$config_file")"

  if [[ ! -f "$config_file" ]]; then
    printf '{"mcpServers":{"adb-link":%s}}\n' "$entry" > "$config_file"
    log "  Created: ${config_file}"
    return
  fi

  if grep -q '"adb-link"' "$config_file" 2>/dev/null; then
    log "  adb-link already registered in ${config_file} — updating entry."
  fi

  if command -v python3 >/dev/null 2>&1 || command -v python >/dev/null 2>&1; then
    local py
    py="$(command -v python3 2>/dev/null || command -v python)"
    "$py" -c "
import json, sys
with open('$config_file', 'r') as f:
    config = json.load(f)
if 'mcpServers' not in config:
    config['mcpServers'] = {}
config['mcpServers']['adb-link'] = $entry
with open('$config_file', 'w') as f:
    json.dump(config, f, indent=2)
    f.write('\n')
"
    log "  Updated: ${config_file}"
  else
    warn "  Cannot auto-merge without Python. Please manually add to ${config_file}:"
    printf '  {"mcpServers":{"adb-link":%s}}\n' "$entry"
  fi
}

register_claude() {
  local entry="$1"
  log "  Registering MCP for Claude Desktop..."
  merge_mcp_config "$(agent_config_path claude)" "$entry"
}

register_cursor() {
  local entry="$1"
  log "  Registering MCP for Cursor..."
  merge_mcp_config "$(agent_config_path cursor)" "$entry"
}

register_windsurf() {
  local entry="$1"
  log "  Registering MCP for Windsurf..."
  merge_mcp_config "$(agent_config_path windsurf)" "$entry"
}

register_qoder_stdio() {
  log "  Registering MCP for Qoder CLI (stdio)..."
  if [[ "$DRY_RUN" == true ]]; then
    log "  [dry-run] Would run: qoder mcp add adb-link -- adb-link run-mcp"
    return
  fi
  if command -v qoder >/dev/null 2>&1; then
    qoder mcp add adb-link -- adb-link run-mcp 2>/dev/null || \
      warn "  Could not auto-register via qoder CLI. Please add manually."
  else
    warn "  qoder CLI not found. Please register manually:"
    warn "  qoder mcp add adb-link -- adb-link run-mcp"
  fi
}

register_qoder_remote() {
  local url="$1"
  log "  Registering MCP for Qoder CLI (remote)..."
  if [[ "$DRY_RUN" == true ]]; then
    log "  [dry-run] Would run: qoder mcp add adb-link --url $url"
    return
  fi
  if command -v qoder >/dev/null 2>&1; then
    qoder mcp add adb-link --url "$url" 2>/dev/null || \
      warn "  Could not auto-register via qoder CLI. Please add manually."
  else
    warn "  qoder CLI not found. Please register manually."
  fi
}

register_mcp() {
  local mode_type="$1"
  local agent entry
  agent="$(detect_agent)"

  if [[ "$mode_type" == "stdio" ]]; then
    entry='{"command":"adb-link","args":["run-mcp"]}'
    log "Registering MCP stdio (target: ${agent})..."
  else
    entry="{\"url\":\"${REMOTE_URL}\"}"
    log "Registering remote MCP (target: ${agent}, url: ${REMOTE_URL})..."
  fi

  case "$agent" in
    claude)   register_claude "$entry" ;;
    cursor)   register_cursor "$entry" ;;
    windsurf) register_windsurf "$entry" ;;
    qoder)
      if [[ "$mode_type" == "stdio" ]]; then
        register_qoder_stdio
      else
        register_qoder_remote "$REMOTE_URL"
      fi
      ;;
    all)
      register_claude "$entry"
      register_cursor "$entry"
      register_windsurf "$entry"
      if [[ "$mode_type" == "stdio" ]]; then
        register_qoder_stdio
      else
        register_qoder_remote "$REMOTE_URL"
      fi
      ;;
    *) die "Unknown agent: ${agent}. Use --agent claude|cursor|windsurf|qoder|all" ;;
  esac
}

# ---------- Verify ----------
verify_binary() {
  log "Verifying installation..."

  if [[ "$DRY_RUN" == true ]]; then
    log "  [dry-run] Would verify: adb-link version"
    return
  fi

  if command -v "$BIN_NAME" >/dev/null 2>&1; then
    log "  $(adb-link version)"
  else
    warn "  adb-link not found in PATH. You may need to restart your terminal or add ${LINK_DIR} to PATH."
  fi
}

# ---------- Main ----------
log "adb-link MCP Setup"
log "==================="

select_mode

case "$SETUP_MODE" in
  cli)
    log "Mode: CLI only"
    install_binary
    verify_binary
    log ""
    log "Setup complete!"
    log "adb-link is installed. Use it directly:"
    log "  adb-link version"
    log "  adb-link run-all   # Start API + MCP HTTP server"
    log "  adb-link run-mcp   # Start MCP stdio server"
    ;;

  remote)
    log "Mode: Remote MCP"
    ask_remote_url
    register_mcp "remote"
    log ""
    log "Setup complete!"
    log "Your Agent is configured to connect to: ${REMOTE_URL}"
    log "No local binary needed — restart your Agent to apply."
    ;;

  stdio)
    log "Mode: Local MCP stdio"
    install_binary
    create_config
    register_mcp "stdio"
    verify_binary
    log ""
    log "Setup complete!"
    log "Next steps:"
    log "  1. Edit ~/.adb-link/conf/mcp_stdio.yaml to add your databases"
    log "  2. Restart your Agent (Claude Desktop / Cursor / Windsurf / Qoder)"
    log "  3. Try calling list_datasources in your Agent"
    ;;
esac
