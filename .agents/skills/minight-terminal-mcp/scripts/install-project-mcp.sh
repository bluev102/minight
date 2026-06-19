#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
EXPECTED_SKILL_SCRIPTS="${PROJECT_ROOT}/.agents/skills/minight-terminal-mcp/scripts"
SERVER_DIR="${SKILL_DIR}/server"
BIN_DIR="${PROJECT_ROOT}/.minight-terminal/bin"
BINARY="${BIN_DIR}/minight-terminal"
MCP_CONFIG="${PROJECT_ROOT}/.cursor/mcp.json"
MCP_DIR="${PROJECT_ROOT}/.cursor"

if [[ "${SCRIPT_DIR}" != "${EXPECTED_SKILL_SCRIPTS}" ]]; then
  echo "ERROR: This installer must run from a project-level skill install." >&2
  echo "Expected path: ${EXPECTED_SKILL_SCRIPTS}" >&2
  echo "Got path: ${SCRIPT_DIR}" >&2
  echo "Install the skill into your project with:" >&2
  echo "  npx skills add bluev102/minight --skill minight-terminal-mcp -a cursor -y" >&2
  exit 1
fi

if [[ ! -f "${SERVER_DIR}/go.mod" ]]; then
  echo "ERROR: Bundled MCP server source missing at ${SERVER_DIR}" >&2
  echo "If you are developing minight locally, run:" >&2
  echo "  bash .agents/skills/minight-terminal-mcp/scripts/sync-server.sh" >&2
  exit 1
fi

resolve_go() {
  if command -v go >/dev/null 2>&1; then
    command -v go
    return 0
  fi
  if [[ -x "${PROJECT_ROOT}/.tools/go/bin/go" ]]; then
    echo "${PROJECT_ROOT}/.tools/go/bin/go"
    return 0
  fi
  return 1
}

echo "Installing minight-terminal MCP config for project"
echo "Project root: ${PROJECT_ROOT}"
echo "Server source: ${SERVER_DIR}"

GO_BIN="$(resolve_go || true)"
if [[ -z "${GO_BIN}" ]]; then
  echo "ERROR: Go not found. Install Go 1.23+ from https://go.dev/dl/" >&2
  exit 1
fi

echo "Using Go: ${GO_BIN}"
mkdir -p "${BIN_DIR}"
(
  cd "${SERVER_DIR}"
  "${GO_BIN}" build -o "${BINARY}" .
)

chmod +x "${BINARY}"
echo "Built binary: ${BINARY}"

mkdir -p "${MCP_DIR}"

python3 - <<'PY' "${MCP_CONFIG}" "${BINARY}"
import json
import os
import sys

config_path, binary_path = sys.argv[1:3]
data = {"mcpServers": {}}

if os.path.exists(config_path):
    with open(config_path, "r", encoding="utf-8") as f:
        loaded = json.load(f)
    if isinstance(loaded, dict):
        data = loaded

servers = data.setdefault("mcpServers", {})
servers["minight-terminal"] = {
    "type": "stdio",
    "command": binary_path,
    "args": [],
    "env": {
        "MAX_TIMEOUT_SECONDS": "300",
    },
}

with open(config_path, "w", encoding="utf-8") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
PY

echo "Updated MCP config: ${MCP_CONFIG}"
echo "Next step: reload MCP in Cursor."
