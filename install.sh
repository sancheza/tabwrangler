#!/bin/bash
#
# install.sh - Automated cross-platform installer for TabWrangler.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/sancheza/tabwrangler/main/install.sh | bash
#   install.sh [-h|--help] [-v|--version] [--dry-run] [--prefix DIR]

VERSION="1.0.12"

set -euo pipefail

# ANSI Color codes per ADR 0020
CLR_RESET="\033[0m"
CLR_GREEN="\033[32m"
CLR_AMBER="\033[38;5;214m"
CLR_RED="\033[31m"

log_info() {
  printf "${CLR_GREEN}INFO:   ${CLR_RESET} %s\n" "$1"
}

log_warn() {
  printf "${CLR_AMBER}WARNING:${CLR_RESET} %s\n" "$1"
}

log_error() {
  printf "${CLR_RED}ERROR:  ${CLR_RESET} %s\n" "$1" >&2
}

show_help() {
  cat << EOF
TabWrangler Installer v${VERSION}

Usage:
  install.sh [OPTIONS]

Options:
  -h, --help      Show this help message and exit
  -v, --version   Show script version and exit
  --dry-run       Print installation actions without modifying the system
  --prefix DIR    Target install directory for the binary (default: ~/.local/bin or /usr/local/bin)
EOF
}

detect_target() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "${os}" in
    darwin)
      case "${arch}" in
        arm64|aarch64) echo "tabwrangler-darwin-arm64" ;;
        x86_64) echo "tabwrangler-darwin-amd64" ;;
        *) log_error "Unsupported macOS architecture: ${arch}"; exit 1 ;;
      esac
      ;;
    linux)
      case "${arch}" in
        x86_64|amd64) echo "tabwrangler-linux-amd64" ;;
        arm64|aarch64) echo "tabwrangler-linux-arm64" ;;
        *) log_error "Unsupported Linux architecture: ${arch}"; exit 1 ;;
      esac
      ;;
    *)
      log_error "Unsupported operating system: ${os}. For Windows, download tabwrangler-windows-amd64.zip."
      exit 1
      ;;
  esac
}

main() {
  local dry_run=0
  local prefix=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help)
        show_help
        exit 0
        ;;
      -v|--version)
        echo "TabWrangler Installer v${VERSION}"
        exit 0
        ;;
      --dry-run)
        dry_run=1
        shift
        ;;
      --prefix)
        prefix="$2"
        shift 2
        ;;
      *)
        log_error "Unknown option: $1"
        show_help
        exit 1
        ;;
    esac
  done

  local binary_name
  binary_name="$(detect_target)"

  # Resolve target install directory
  if [[ -z "${prefix}" ]]; then
    if [[ -d "${HOME}/.local/bin" ]] || [[ ":${PATH}:" == *":${HOME}/.local/bin:"* ]]; then
      prefix="${HOME}/.local/bin"
    elif [[ -w "/usr/local/bin" ]]; then
      prefix="/usr/local/bin"
    else
      prefix="${HOME}/.local/bin"
    fi
  fi

  local download_url="https://github.com/sancheza/tabwrangler/releases/latest/download/${binary_name}"
  local target_path="${prefix}/tabwrangler"

  if [[ ${dry_run} -eq 1 ]]; then
    log_info "[DRY-RUN] Would create directory: ${prefix}"
    log_info "[DRY-RUN] Would download: ${download_url} -> ${target_path}"
    log_info "[DRY-RUN] Would make executable: chmod +x ${target_path}"
    log_info "[DRY-RUN] Would run: ${target_path} setup"
    exit 0
  fi

  log_info "Installing TabWrangler to ${prefix}..."
  mkdir -p "${prefix}"

  log_info "Downloading ${download_url}..."
  if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
    # Authenticated GitHub CLI downloads from both public and private repos
    gh release download -R sancheza/tabwrangler -p "${binary_name}" -D "${prefix}" --clobber
    mv "${prefix}/${binary_name}" "${target_path}"
  elif command -v curl >/dev/null 2>&1; then
    local auth_header=()
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
      auth_header=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
    fi
    curl -fsSL "${auth_header[@]}" "${download_url}" -o "${target_path}"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "${target_path}" "${download_url}"
  else
    log_error "Neither gh, curl, nor wget found in PATH."
    exit 1
  fi

  chmod +x "${target_path}"
  log_info "Installed binary: ${target_path}"

  # Run initial setup to register native messaging manifests for Chrome & Firefox
  log_info "Configuring browser native messaging manifests..."
  "${target_path}" setup

  # Check if install directory is in user's PATH
  if [[ ":${PATH}:" != *":${prefix}:"* ]]; then
    log_warn "Target directory '${prefix}' is not in your current PATH."
    log_warn "Add it by adding this line to your ~/.zshrc or ~/.bashrc:"
    log_warn "  export PATH=\"${prefix}:\$PATH\""
  fi

  log_info "TabWrangler installed successfully!"
  log_info "Run 'tabwrangler' to launch."
}

main "$@"
