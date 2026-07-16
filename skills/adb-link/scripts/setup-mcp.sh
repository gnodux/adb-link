#!/usr/bin/env bash
#
# setup-mcp.sh — One-click install adb-link and register MCP for your Agent.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/skills/adb-link/scripts/setup-mcp.sh | bash
#   bash setup-mcp.sh [--agent claude|cursor|windsurf|qoder|all] [--dry-run]
#
# Flags:
#   --agent   Target agent platform (default: auto-detect, or "all")
#   --dry-run Show what would be done without making changes
#   --help    Show this help message

set -euo pipefail

REPO="gnodux/adb-link"
INSTALL_ROOT="${HOME}/.adb-link"
CONF_DIR="${INSTALL_ROOT}/conf"
LINK_DIR="${HOME}/.local/bin"
BIN_NAME="adb-link"

AGENT_TARGET=""
DRY_RUN=false

log()  { printf '\033[1;34m[adb-link]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[adb-link]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[adb-link]\033[0m %s\n' "$*" >&2; }
die()  { err "$*"; exit 1; }

# ---------- Parse args ----------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --agent)  AGENT_TARGET="$2"; shift 2 ;;
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

# ---------- Step 1: Install binary ----------
install_binary() {
  log "Step 1: Installing adb-link binary..."

  if command -v "$BIN_NAME" >/dev/null 2>&1; then
    local current_ver
    current_ver="$("$BIN_NAME" version 2>/dev/null || echo "unknown")"
    log "adb-link already installed ($current_ver). Will update to latest."
  fi

  if [[ "$DRY_RUN" == true ]]; then
    log "  [dry-run] Would run: curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/install-adb-link.sh | bash"
    return
  fi

  curl -fsSL "https://raw.githubusercontent.com/${REPO}/main/scripts/install-adb-link.sh" | bash
}

# ---------- Step 2: Create default config ----------
create_config() {
  log "Step 2: Setting up configuration..."

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

  # Check for Claude Desktop config
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

  # Check for Cursor
  if [[ -f "${HOME}/.cursor/mcp.json" ]] || [[ -f ".cursor/mcp.json" ]]; then
    echo "cursor"
    return
  fi

  # Check for Windsurf
  local windsurf_config="${HOME}/.codeium/windsurf/mcp_config.json"
  if [[ -f "$windsurf_config" ]]; then
    echo "windsurf"
    return
  fi

  # Check for Qoder
  if [[ -d "${HOME}/.qoder" ]]; then
    echo "qoder"
    return
  fi

  # Default: register for all
  echo "all"
}

# ---------- Step 4: Register MCP ----------
# Merge adb-link into a JSON config file, preserving existing entries.
merge_mcp_config() {
  local config_file="$1"
  local entry='{"command":"adb-link","args":["run-mcp"]}'

  if [[ "$DRY_RUN" == true ]]; then
    log "  [dry-run] Would merge adb-link MCP entry into: ${config_file}"
    return
  fi

  mkdir -p "$(dirname "$config_file")"

  if [[ ! -f "$config_file" ]]; then
    # Create new config file
    printf '{"mcpServers":{"adb-link":%s}}\n' "$entry" > "$config_file"
    log "  Created: ${config_file}"
    return
  fi

  # Check if adb-link already exists
  if grep -q '"adb-link"' "$config_file" 2>/dev/null; then
    log "  adb-link already registered in ${config_file} — skipping."
    return
  fi

  # Merge using Python if available, otherwise use a simple approach
  if command -v python3 >/dev/null 2>&1; then
    python3 -c "
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
  elif command -v python >/dev/null 2>&1; then
    python -c "
import json
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
  log "Registering MCP for Claude Desktop..."
  local os config_file
  os="$(uname -s)"
  case "$os" in
    Darwin) config_file="${HOME}/Library/Application Support/Claude/claude_desktop_config.json" ;;
    Linux)  config_file="${HOME}/.config/Claude/claude_desktop_config.json" ;;
    MINGW*|MSYS*|CYGWIN*|Windows_NT) config_file="${APPDATA}/Claude/claude_desktop_config.json" ;;
    *) die "Unsupported OS for Claude Desktop: $os" ;;
  esac
  merge_mcp_config "$config_file"
}

register_cursor() {
  log "Registering MCP for Cursor..."
  merge_mcp_config "${HOME}/.cursor/mcp.json"
}

register_windsurf() {
  log "Registering MCP for Windsurf..."
  local config_file="${HOME}/.codeium/windsurf/mcp_config.json"
  merge_mcp_config "$config_file"
}

register_qoder() {
  log "Registering MCP for Qoder CLI..."
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

register_mcp() {
  local agent
  agent="$(detect_agent)"
  log "Step 3: Registering MCP (target: ${agent})..."

  case "$agent" in
    claude)   register_claude ;;
    cursor)   register_cursor ;;
    windsurf) register_windsurf ;;
    qoder)    register_qoder ;;
    all)
      register_claude
      register_cursor
      register_windsurf
      register_qoder
      ;;
    *) die "Unknown agent: ${agent}. Use --agent claude|cursor|windsurf|qoder|all" ;;
  esac
}

# ---------- Step 5: Verify ----------
verify() {
  log "Step 4: Verifying installation..."

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
install_binary
create_config
register_mcp
verify
log ""
log "Setup complete!"
log "Next steps:"
log "  1. Edit ~/.adb-link/conf/mcp_stdio.yaml to add your databases"
log "  2. Restart your Agent (Claude Desktop / Cursor / Windsurf / Qoder)"
log "  3. Try calling list_datasources in your Agent"
