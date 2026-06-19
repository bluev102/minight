#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${SKILL_DIR}/../../.." && pwd)"
SERVER_DIR="${SKILL_DIR}/server"

MODE="sync"
if [[ "${1:-}" == "--check" ]]; then
  MODE="check"
elif [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  echo "Usage: sync-server.sh [--check]"
  echo "  (default) Copy MCP server source from repo root into skill/server/"
  echo "  --check   Exit non-zero if bundled server source is stale"
  exit 0
elif [[ -n "${1:-}" ]]; then
  echo "Unknown option: $1" >&2
  exit 1
fi

SOURCE_FILES=(main.go go.mod go.sum)
SOURCE_DIRS=(internal)

require_sources() {
  for file in "${SOURCE_FILES[@]}"; do
    if [[ ! -f "${REPO_ROOT}/${file}" ]]; then
      echo "ERROR: missing source file: ${REPO_ROOT}/${file}" >&2
      exit 1
    fi
  done
  for dir in "${SOURCE_DIRS[@]}"; do
    if [[ ! -d "${REPO_ROOT}/${dir}" ]]; then
      echo "ERROR: missing source directory: ${REPO_ROOT}/${dir}" >&2
      exit 1
    fi
  done
}

sync_sources() {
  require_sources
  mkdir -p "${SERVER_DIR}"
  for file in "${SOURCE_FILES[@]}"; do
    cp "${REPO_ROOT}/${file}" "${SERVER_DIR}/${file}"
  done
  rm -rf "${SERVER_DIR}/internal"
  cp -R "${REPO_ROOT}/internal" "${SERVER_DIR}/internal"
  echo "Synced MCP server source to ${SERVER_DIR}"
}

collect_snapshot() {
  local root="$1"
  find "${root}" -type f \( -name 'main.go' -o -name 'go.mod' -o -name 'go.sum' -o -name '*.go' \) | while IFS= read -r file; do
    rel="${file#"${root}/"}"
    case "${rel}" in
      main.go|go.mod|go.sum|internal/*) ;;
      *) continue ;;
    esac
    sha256sum "${file}" | awk -v rel="${rel}" '{print rel "|" $1}'
  done | sort
}

check_sources() {
  require_sources
  if [[ ! -d "${SERVER_DIR}" ]]; then
    echo "ERROR: bundled server directory missing: ${SERVER_DIR}" >&2
    echo "Run: bash .agents/skills/minight-terminal-mcp/scripts/sync-server.sh" >&2
    exit 1
  fi

  tmp="$(mktemp -d)"
  trap 'rm -rf "${tmp}"' EXIT

  collect_snapshot "${REPO_ROOT}" > "${tmp}/root.txt"
  collect_snapshot "${SERVER_DIR}" > "${tmp}/server.txt"

  if diff -u "${tmp}/root.txt" "${tmp}/server.txt" >/dev/null; then
    echo "PASS: bundled server source is in sync"
    exit 0
  fi

  echo "ERROR: bundled server source is stale" >&2
  echo "Run: bash .agents/skills/minight-terminal-mcp/scripts/sync-server.sh" >&2
  diff -u "${tmp}/root.txt" "${tmp}/server.txt" >&2 || true
  exit 1
}

case "${MODE}" in
  sync) sync_sources ;;
  check) check_sources ;;
esac
