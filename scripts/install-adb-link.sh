#!/usr/bin/env bash
#
# install-adb-link.sh
# Install the latest release of github.com/gnodux/adb-link
#
# Layout:
#   ~/.adb-link/            install root
#   ~/.adb-link/bin/        unpacked release binaries
#   ~/.adb-link/cache/      downloaded archives
#   ~/.local/bin/adb-link   symlink -> ~/.adb-link/bin/adb-link
#

set -euo pipefail

REPO="gnodux/adb-link"
INSTALL_ROOT="${HOME}/.adb-link"
BIN_DIR="${INSTALL_ROOT}/bin"
CACHE_DIR="${INSTALL_ROOT}/cache"
LINK_DIR="${HOME}/.local/bin"
LINK_PATH="${LINK_DIR}/adb-link"
BIN_NAME="adb-link"
EXE_SUFFIX=""

log()  { printf '\033[1;34m[adb-link]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[adb-link]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[adb-link]\033[0m %s\n' "$*" >&2; exit 1; }

require() {
  command -v "$1" >/dev/null 2>&1 || die "Missing required command: $1"
}

require curl
require tar
require uname
require ln

# ---------- Detect OS / Arch ----------
detect_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"

  case "$os" in
    Darwin) OS_NAME="darwin" ;;
    Linux)  OS_NAME="linux" ;;
    MINGW*|MSYS*|CYGWIN*|Windows_NT)
      OS_NAME="windows"
      EXE_SUFFIX=".exe"
      ;;
    *) die "Unsupported OS: $os" ;;
  esac

  case "$arch" in
    x86_64|amd64) ARCH_NAME="amd64" ;;
    arm64|aarch64) ARCH_NAME="arm64" ;;
    *) die "Unsupported arch: $arch" ;;
  esac

  log "Detected platform: ${OS_NAME}/${ARCH_NAME}"
}

# ---------- Resolve latest release ----------
fetch_latest_release() {
  local api="https://api.github.com/repos/${REPO}/releases/latest"
  log "Querying latest release: ${api}"

  local headers=()
  if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    headers+=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
  fi

  RELEASE_JSON="$(curl -fsSL ${headers[@]+"${headers[@]}"} \
    -H 'Accept: application/vnd.github+json' \
    "$api")" || die "Failed to query GitHub API. Check network or set GITHUB_TOKEN."

  TAG_NAME="$(printf '%s' "$RELEASE_JSON" \
    | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  [[ -n "$TAG_NAME" ]] || die "Cannot parse tag_name from API response."
  log "Latest release: ${TAG_NAME}"
}

# ---------- Pick a matching asset ----------
pick_asset() {
  # Extract all browser_download_url entries.
  local urls
  urls="$(printf '%s' "$RELEASE_JSON" \
    | grep -Eo '"browser_download_url":[[:space:]]*"[^"]+"' \
    | sed -E 's/"browser_download_url":[[:space:]]*"([^"]+)"/\1/')"

  [[ -n "$urls" ]] || die "No release assets found for ${TAG_NAME}."

  # Try to match os + arch (with common synonyms).
  local os_alts="${OS_NAME}"
  local arch_alts="${ARCH_NAME}"
  case "$OS_NAME" in
    darwin)  os_alts="darwin|macos|mac|osx" ;;
    linux)   os_alts="linux" ;;
    windows) os_alts="windows|win64|win32|win" ;;
  esac
  case "$ARCH_NAME" in
    amd64) arch_alts="amd64|x86_64|x64" ;;
    arm64) arch_alts="arm64|aarch64" ;;
  esac

  local archive_re='\.(tar\.gz|tgz|zip)$'
  if [[ "$OS_NAME" == "windows" ]]; then
    archive_re='\.(zip|tar\.gz|tgz|exe)$'
  fi

  # First try archives matching os+arch.
  ASSET_URL="$(printf '%s\n' "$urls" \
    | grep -Ei "(${os_alts})" \
    | grep -Ei "(${arch_alts})" \
    | grep -Ei "$archive_re" \
    | head -n1 || true)"

  # Fallback: raw binary (no archive extension), still requires os+arch match.
  if [[ -z "$ASSET_URL" ]]; then
    ASSET_URL="$(printf '%s\n' "$urls" \
      | grep -Ei "(${os_alts})" \
      | grep -Ei "(${arch_alts})" \
      | grep -Eiv '\.(sha256|sha512|md5|asc|sig|txt|json|yml|yaml)$' \
      | head -n1 || true)"
  fi

  if [[ -z "$ASSET_URL" ]]; then
    warn "No asset matched ${OS_NAME}/${ARCH_NAME}. Available assets:"
    printf '  %s\n' $urls >&2
    die "Please open an issue or download manually."
  fi

  ASSET_NAME="$(basename "$ASSET_URL")"
  log "Selected asset: ${ASSET_NAME}"
}

# ---------- Download + extract ----------
download_and_extract() {
  mkdir -p "$BIN_DIR" "$CACHE_DIR"
  local archive_path="${CACHE_DIR}/${ASSET_NAME}"

  log "Downloading -> ${archive_path}"
  curl -fSL --retry 3 --retry-delay 2 -o "${archive_path}.part" "$ASSET_URL"
  mv "${archive_path}.part" "${archive_path}"

  local stage
  stage="$(mktemp -d)"
  trap 'rm -rf "$stage"' RETURN

  log "Extracting to staging: ${stage}"
  case "$ASSET_NAME" in
    *.tar.gz|*.tgz) tar -xzf "$archive_path" -C "$stage" ;;
    *.zip)
      if command -v unzip >/dev/null 2>&1; then
        unzip -q "$archive_path" -d "$stage"
      elif command -v tar >/dev/null 2>&1 && tar --help 2>&1 | grep -q -- '--format'; then
        # bsdtar (default tar on Windows 10+/macOS) can read zip archives.
        tar -xf "$archive_path" -C "$stage"
      else
        die "Need 'unzip' or bsdtar to extract zip archives."
      fi
      ;;
    *)
      # Treat as a raw binary: copy into staging under the canonical name.
      log "Asset is a raw binary, no extraction needed."
      cp "$archive_path" "${stage}/${BIN_NAME}${EXE_SUFFIX}"
      chmod +x "${stage}/${BIN_NAME}${EXE_SUFFIX}" 2>/dev/null || true
      ;;
  esac

  # Locate the actual binary inside the staging dir.
  local found
  found="$(find "$stage" -type f \( -name "${BIN_NAME}${EXE_SUFFIX}" -o -name "${BIN_NAME}.exe" -o -name "${BIN_NAME}" \) | head -n1 || true)"
  if [[ -z "$found" ]]; then
    # Fallback: any executable file
    found="$(find "$stage" -type f -perm -u+x | head -n1 || true)"
  fi
  [[ -n "$found" ]] || die "Could not locate '${BIN_NAME}' inside the archive."

  log "Installing binary -> ${BIN_DIR}/${BIN_NAME}${EXE_SUFFIX}"
  install -m 0755 "$found" "${BIN_DIR}/${BIN_NAME}${EXE_SUFFIX}"

  # Copy example config files to ~/.adb-link/conf/ if not already present.
  local src_dir
  src_dir="$(dirname "$found")"
  local conf_src="${src_dir}/conf"
  local conf_dst="${INSTALL_ROOT}/conf"
  if [[ -d "$conf_src" ]]; then
    mkdir -p "$conf_dst"
    for f in "$conf_src"/*.example; do
      [[ -f "$f" ]] || continue
      local base
      base="$(basename "$f")"
      if [[ ! -f "${conf_dst}/${base}" ]]; then
        cp "$f" "${conf_dst}/${base}"
        log "Installed config example: ${conf_dst}/${base}"
      fi
    done
  fi

  printf '%s\n' "$TAG_NAME" > "${INSTALL_ROOT}/VERSION"
}

# ---------- Symlink (or shim on Windows) ----------
create_symlink() {
  mkdir -p "$LINK_DIR"
  local target="${BIN_DIR}/${BIN_NAME}${EXE_SUFFIX}"
  local link="${LINK_PATH}${EXE_SUFFIX}"

  if [[ -L "$link" || -e "$link" ]]; then
    log "Removing existing ${link}"
    rm -f "$link"
  fi

  if [[ "$OS_NAME" == "windows" ]]; then
    # On MSYS/Git Bash, ln -s often produces a copy or fails without
    # Developer Mode. Try a real symlink first, fall back to a .cmd shim.
    if MSYS=winsymlinks:nativestrict ln -s "$target" "$link" 2>/dev/null; then
      log "Native symlink created: ${link} -> ${target}"
    else
      warn "Native symlink unavailable. Writing .cmd shim instead."
      local shim="${LINK_DIR}/${BIN_NAME}.cmd"
      # Convert MSYS path to a Windows-style path for cmd.
      local win_target
      if command -v cygpath >/dev/null 2>&1; then
        win_target="$(cygpath -w "$target")"
      else
        win_target="$target"
      fi
      printf '@echo off\r\n"%s" %%*\r\n' "$win_target" > "$shim"
      log "Shim created: ${shim} -> ${win_target}"
    fi
  else
    ln -s "$target" "$link"
    log "Symlink created: ${link} -> ${target}"
  fi
}

# ---------- PATH hint ----------
path_hint() {
  case ":$PATH:" in
    *":${LINK_DIR}:"*) ;;
    *)
      warn "${LINK_DIR} is not in your PATH. Add this line to your shell rc:"
      printf '\n  export PATH="%s:$PATH"\n\n' "$LINK_DIR" >&2
      ;;
  esac
}

# ---------- Main ----------
detect_platform
fetch_latest_release
pick_asset
download_and_extract
create_symlink
path_hint

log "Done. Installed adb-link ${TAG_NAME}."
log "Try: adb-link --version"
