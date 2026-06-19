# Minight Terminal MCP Reference

## Server

- Name: `minight-terminal`
- Transport: stdio MCP server
- Binary: `minight-terminal` (build with `go build -o minight-terminal .`)
- Project MCP config: `.cursor/mcp.json`

Example registration:

```json
{
  "mcpServers": {
    "minight-terminal": {
      "type": "stdio",
      "command": "/absolute/path/to/minight-terminal",
      "args": [],
      "env": {
        "MAX_TIMEOUT_SECONDS": "300"
      }
    }
  }
}
```

## Tools

### run_command

Execute a shell command in a session.

Input:
- `command` (required): shell command string
- `session_id` (optional): defaults to `default`
- `timeout` (optional): seconds, default 30, capped by `MAX_TIMEOUT_SECONDS`
- `cwd` (optional): overrides session cwd for this call
- `verbose` (optional): include debug metadata

Compact response fields:
- `stdout`, `stderr`
- `return_code`
- `timed_out`
- `current_cwd`
- `truncated`

Verbose-only fields:
- `duration_ms`
- `stdout_omitted_bytes`
- `stderr_omitted_bytes`
- `session_id`
- `env_changed_count`

Error field:
- `error`: validation, safety rejection, or execution error

### get_session

Return session location.

Input:
- `session_id` (optional): defaults to `default`

Response:
- `session_id`
- `current_cwd`

Environment values are intentionally not returned.

### kill_session

Reset/delete in-memory session state.

Input:
- `session_id` (optional): defaults to `default`

Response:
- `session_id`
- `reset`: true

## Session Semantics

- Missing `session_id` uses `default`
- Unknown sessions auto-create at user home directory
- Each command runs in a short-lived shell via `$SHELL -lc`
- After each command, server captures final cwd/env via internal trailer
- Session updates even on non-zero exit codes
- On timeout, process group is killed and session state is not updated

## Output Processing

- ANSI escape codes are stripped
- Long stdout/stderr uses head/tail truncation (~3000 chars)
- Truncated output includes omitted byte counts when `verbose: true`

## Security Model

Trusted local tool with best-effort guardrails. Not a sandbox.

Rejected patterns include obvious destructive commands such as:
- `rm -rf /`
- fork bomb patterns
- filesystem formatting (`mkfs`)
- destructive `dd`
- shutdown/reboot commands

Rejections return valid JSON with short `error` reason.

## Limitations

- No persistent PTY or interactive terminal emulation
- No streaming output
- No background job management
- Env persistence uses post-command trailer capture, not a long-lived shell
- Aliases/functions may differ from interactive terminal behavior
- MCP shell PATH may be narrower than Cursor built-in terminal PATH

## Configuration

Environment variables for MCP server process:
- `MAX_TIMEOUT_SECONDS`: max allowed timeout per command (default server max 300 unless overridden)
- `SHELL`: shell binary (fallback `/bin/sh`)

Recommended: set `PATH` in MCP env if tools like `go`, `node`, or `npm` must be available without session export.

## Verification Commands

```bash
bash .agents/skills/minight-terminal-mcp/scripts/verify-minight-terminal.sh
go test ./...
go test ./e2e
go build -o minight-terminal .
```
