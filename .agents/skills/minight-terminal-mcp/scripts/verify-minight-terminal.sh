#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
EXPECTED_SKILL_SCRIPTS="${PROJECT_ROOT}/.agents/skills/minight-terminal-mcp/scripts"
SERVER_DIR="${SKILL_DIR}/server"
DEFAULT_BINARY="${PROJECT_ROOT}/.minight-terminal/bin/minight-terminal"
DEFAULT_BINARY_EXE="${PROJECT_ROOT}/.minight-terminal/bin/minight-terminal.exe"
LEGACY_BINARY="${PROJECT_ROOT}/minight-terminal"
MCP_CONFIG="${PROJECT_ROOT}/.cursor/mcp.json"

pass=0
warn=0
fail=0

pass_line() {
  echo "PASS: $1"
  pass=$((pass + 1))
}

warn_line() {
  echo "WARN: $1"
  warn=$((warn + 1))
}

fail_line() {
  echo "FAIL: $1"
  fail=$((fail + 1))
}

resolve_python() {
  if command -v python3 >/dev/null 2>&1; then
    echo "python3"
    return 0
  fi
  if command -v python >/dev/null 2>&1; then
    echo "python"
    return 0
  fi
  if command -v py >/dev/null 2>&1; then
    echo "py -3"
    return 0
  fi
  return 1
}

echo "Minight Terminal MCP verification"
echo "Project root: ${PROJECT_ROOT}"
echo

if [[ "${SCRIPT_DIR}" != "${EXPECTED_SKILL_SCRIPTS}" ]]; then
  warn_line "Running outside project-level skill path (${SCRIPT_DIR})"
  warn_line "Install with: npx skills add bluev102/minight --skill minight-terminal-mcp -a cursor -y"
fi

if [[ -f "${SERVER_DIR}/go.mod" ]]; then
  pass_line "Bundled server source exists: ${SERVER_DIR}/go.mod"
else
  fail_line "Bundled server source missing: ${SERVER_DIR}/go.mod"
fi

if [[ -x "${DEFAULT_BINARY_EXE}" ]]; then
  pass_line "Project binary exists: ${DEFAULT_BINARY_EXE}"
elif [[ -x "${DEFAULT_BINARY}" ]]; then
  if [[ "${OSTYPE:-}" == msys* || "${OSTYPE:-}" == cygwin* || -n "${MSYSTEM:-}" ]]; then
    warn_line "Binary missing .exe suffix on Windows: ${DEFAULT_BINARY}"
    warn_line "Re-run install-project-mcp.sh to build minight-terminal.exe"
  else
    pass_line "Project binary exists: ${DEFAULT_BINARY}"
  fi
elif [[ -x "${LEGACY_BINARY}" ]]; then
  warn_line "Legacy repo-root binary exists: ${LEGACY_BINARY}"
else
  fail_line "Binary missing: ${DEFAULT_BINARY_EXE} (run install-project-mcp.sh)"
fi

if [[ -f "${MCP_CONFIG}" ]]; then
  pass_line "Project MCP config exists: ${MCP_CONFIG}"
else
  fail_line "Project MCP config missing: ${MCP_CONFIG}"
fi

PYTHON_BIN="$(resolve_python || true)"
if [[ -f "${MCP_CONFIG}" && -n "${PYTHON_BIN}" ]]; then
  read -r line < <(
    ${PYTHON_BIN} - <<'PY' "${MCP_CONFIG}" "${DEFAULT_BINARY}" "${DEFAULT_BINARY_EXE}" "${LEGACY_BINARY}"
import json
import sys

config_path, default_binary, default_binary_exe, legacy_binary = sys.argv[1:5]
with open(config_path, "r", encoding="utf-8") as f:
    data = json.load(f)

entry = data.get("mcpServers", {}).get("minight-terminal")
if not entry:
    print("missing|no|no|no|no")
    raise SystemExit(0)

command = entry.get("command", "")
env = entry.get("env", {}) or {}
backend = env.get("MINIGHT_BACKEND", "")
shell = env.get("MINIGHT_SHELL", "")

if command == default_binary_exe:
    status = "default_exe"
elif command == default_binary:
    status = "default"
elif command == legacy_binary:
    status = "legacy"
else:
    status = "other"

print(f"{command}|{status}|yes|{backend}|{shell}")
PY
  )
  IFS='|' read -r command_path match_status has_entry backend shell <<< "${line}"

  if [[ "${has_entry}" != "yes" ]]; then
    fail_line "MCP config has no minight-terminal entry"
  elif [[ "${match_status}" == "default_exe" ]]; then
    pass_line "MCP command path matches project binary: ${command_path}"
  elif [[ "${match_status}" == "default" ]]; then
    if [[ "${OSTYPE:-}" == msys* || "${OSTYPE:-}" == cygwin* || -n "${MSYSTEM:-}" ]]; then
      warn_line "MCP config uses non-.exe binary path on Windows (${command_path})"
    else
      pass_line "MCP command path matches project binary: ${command_path}"
    fi
  elif [[ "${match_status}" == "legacy" ]]; then
    warn_line "MCP config uses legacy repo-root binary path (${command_path})"
    warn_line "Re-run install-project-mcp.sh to migrate to .minight-terminal/bin/"
  else
    warn_line "MCP command path differs from expected project binary (${command_path})"
  fi

  if [[ "${OSTYPE:-}" == msys* || "${OSTYPE:-}" == cygwin* || -n "${MSYSTEM:-}" ]]; then
    if [[ -z "${backend}" ]]; then
      warn_line "MCP env missing MINIGHT_BACKEND (recommended: posix on Git Bash)"
    elif [[ "${backend}" == "posix" ]]; then
      pass_line "MCP env MINIGHT_BACKEND=posix"
    fi
    if [[ -z "${shell}" ]]; then
      warn_line "MCP env missing MINIGHT_SHELL (Cursor may inherit invalid SHELL=/bin/sh)"
    else
      pass_line "MCP env MINIGHT_SHELL is set"
    fi
  fi
elif [[ -f "${MCP_CONFIG}" ]]; then
  warn_line "Python not found; skipping MCP config content checks"
fi

if command -v go >/dev/null 2>&1; then
  pass_line "Go toolchain available in PATH"
elif [[ -x "${PROJECT_ROOT}/.tools/go/bin/go" ]]; then
  warn_line "Go not in PATH, but project-local Go exists at .tools/go/bin/go"
else
  warn_line "Go toolchain not found (build may fail)"
fi

echo
echo "Summary: pass=${pass} warn=${warn} fail=${fail}"
if [[ "${fail}" -gt 0 ]]; then
  echo "Next step: bash .agents/skills/minight-terminal-mcp/scripts/install-project-mcp.sh"
  exit 1
fi

echo "Next step: reload MCP in Cursor if config changed recently."
exit 0
