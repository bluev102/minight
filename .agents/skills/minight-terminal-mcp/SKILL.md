---
name: minight-terminal-mcp
description: Install, verify, and use the minight-terminal MCP server for session-aware shell commands with persistent cwd and environment. Use when the user mentions minight-terminal, MCP terminal, terminal MCP, session-aware shell, persistent cwd/env, or comparing MCP terminal with Cursor built-in terminal.
license: MIT
compatibility: Requires Cursor MCP, Go 1.23+, local host shell access, and stdio MCP transport.
metadata:
  version: "0.1.0"
  repository: https://github.com/bluev102/minight
---

# Minight Terminal MCP

## Quick Start

1. Verify setup:
   ```bash
   bash .agents/skills/minight-terminal-mcp/scripts/verify-minight-terminal.sh
   ```
2. If missing, install project MCP config:
   ```bash
   bash .agents/skills/minight-terminal-mcp/scripts/install-project-mcp.sh
   ```
3. Reload MCP in Cursor after install.
4. Read MCP tool descriptors before calling tools.
5. Use `run_command` with a stable `session_id` per workflow.

## When To Use MCP vs Built-in Terminal

**Prefer `minight-terminal` MCP when:**
- Commands need structured JSON output (`return_code`, `timed_out`, `truncated`)
- Session must persist `cwd` across multiple commands
- Session must persist env (`export PATH=...` then later commands)
- Output should be token-friendly (ANSI stripped, head/tail truncation)
- Timeout control is needed per command
- Safety guardrails for obviously dangerous commands are acceptable

**Prefer built-in terminal when:**
- Full inherited shell PATH is required immediately (nvm, aliases, local tools)
- Interactive behavior, full raw output, or long unbounded logs are needed
- One-off repo setup outside MCP session context

## Core Workflow

1. Choose a stable `session_id` (example: `project-build`, `debug-session`).
2. Optionally set initial `cwd` on first command.
3. Run commands with `run_command`.
4. Use `get_session` to inspect current cwd.
5. Use `kill_session` to reset state when done.

## Tool Usage Rules

- Required for `run_command`: `command`
- Optional: `session_id` (default `default`), `timeout` (seconds), `cwd`, `verbose`
- Use `verbose: true` only when debugging timing, truncation, or env changes
- Do not expect env values in responses; only cwd is exposed via `get_session`
- On timeout, session state is not updated
- Treat safety guardrails as best-effort, not a sandbox

## Common Patterns

**Inspect location**
```json
{"command":"pwd","session_id":"main","cwd":"/path/to/project"}
```

**Persist cwd**
```json
{"command":"cd backend && pwd","session_id":"main"}
{"command":"go test ./...","session_id":"main","timeout":120}
```

**Persist PATH once per session**
```json
{"command":"export PATH=/path/to/go/bin:$PATH","session_id":"main"}
{"command":"go test ./...","session_id":"main","timeout":120}
```

**Reset session**
```json
{"session_id":"main"}
```
Call `kill_session` with the above args.

## Response Handling

Parse JSON from tool result text. Key fields:
- `return_code`: command exit code
- `timed_out`: true if killed by timeout
- `truncated`: true if stdout/stderr was shortened
- `current_cwd`: session cwd after command
- `error`: validation/safety/execution error message

If `truncated` is true and details are missing, rerun with narrower command or `verbose: true`.

## Additional Resources

- Tool contract and behavior: [reference.md](reference.md)
- Workflows and comparisons: [examples.md](examples.md)
- Packaging/copy instructions: [README.md](README.md)
