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
        "DEFAULT_TIMEOUT_SECONDS": "60",
        "MAX_TIMEOUT_SECONDS": "300",
        "MINIGHT_BACKEND": "auto"
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
- `timeout` (optional): seconds; defaults to `DEFAULT_TIMEOUT_SECONDS` (30 if unset); capped by `MAX_TIMEOUT_SECONDS`
- `cwd` (optional): overrides session cwd for this call
- `verbose` (optional): include debug metadata
- `fail_on_any_error` (optional): bash ERR-trap mode; sets `had_failure` for earlier failures in `;` chains
- `pipefail` (optional): bash `set -o pipefail`
- `strip_crlf` (optional): override server default CRLF stripping

Compact response fields:
- `stdout`, `stderr`
- `return_code` — shell exit code of the full command string
- `had_failure` — true when any segment failed or exit code was non-zero
- `timed_out`
- `current_cwd`
- `cwd_persisted` — session cwd matched shell-reported cwd
- `truncated`

Verbose-only fields:
- `duration_ms`
- `stdout_omitted_bytes`, `stderr_omitted_bytes`
- `stdout_total_bytes`, `stderr_total_bytes`
- `session_id`
- `env_changed_count`
- `environment_warnings` — e.g. WSL drvfs slow-mount hints
- `suggested_timeout` — on timeout, suggested retry timeout

Error field:
- `error`: validation, safety rejection, or execution error

### get_session

Return session metadata.

Input:
- `session_id` (optional): defaults to `default`

Response:
- `session_id`, `current_cwd`
- `env_key_count`, `last_command`, `background_jobs`
- `last_return_code`, `last_had_failure`

Environment values are intentionally not returned.

### list_sessions

List active sessions with the same safe metadata as `get_session`.

### kill_session

Reset/delete in-memory session state.

Input:
- `session_id` (optional): defaults to `default`
- `terminate_background_jobs` (optional): kill tracked background PIDs

Response:
- `session_id`, `reset`
- `background_jobs_killed` when termination requested

## Session Semantics

- Missing `session_id` uses `default`
- Unknown sessions auto-create at user home directory
- Each command runs in a short-lived shell via `posix` or `windows` backend
- After each command, server captures final cwd/env via internal trailer with explicit `__MINIGHT_CWD=` marker
- Session updates even on non-zero exit codes
- On timeout, process group is killed and session state is not updated

## Windows Guidance

| Setup | When to use |
|-------|-------------|
| `MINIGHT_BACKEND=windows` (native `.exe`) | Windows repos on local drives; fastest git/filesystem |
| WSL bridge via `wsl.exe` | Linux toolchain required; accept `/mnt/<drive>/` paths |
| Built-in Cursor terminal | Interactive, aliases, long logs, heavy Windows PATH |

Under WSL, `/e/...` paths are auto-normalized to `/mnt/e/...` when `MINIGHT_NORMALIZE_WSL_PATHS=true`.

## Output Processing

- ANSI escape codes are stripped
- CRLF is normalized by default (`MINIGHT_STRIP_CRLF=true`)
- Long stdout/stderr uses head/tail truncation (~3000 chars)
- Truncated output includes omitted/total byte counts when `verbose: true`

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
- Background job tracking is best-effort; not all `&` jobs may be captured
- Env persistence uses post-command trailer capture, not a long-lived shell
- Aliases/functions may differ from interactive terminal behavior
- MCP shell PATH may be narrower than Cursor built-in terminal PATH

## Configuration

Environment variables for MCP server process:
- `DEFAULT_TIMEOUT_SECONDS`: default per-command timeout (default 30)
- `MAX_TIMEOUT_SECONDS`: max allowed timeout per command (default 300)
- `MINIGHT_BACKEND`: `auto`, `posix`, or `windows`
- `MINIGHT_SHELL`: shell binary override
- `MINIGHT_STRIP_CRLF`: default CRLF stripping (default true)
- `MINIGHT_NORMALIZE_WSL_PATHS`: WSL path shorthand normalization (default true)
- `MINIGHT_OUTPUT_LIMIT`: truncation limit in characters (default 3000)

Recommended: set `PATH` in MCP env if tools like `go`, `node`, or `npm` must be available without session export.

## Verification Commands

```bash
bash .agents/skills/minight-terminal-mcp/scripts/verify-minight-terminal.sh
go test ./...
go test ./e2e
go build -o minight-terminal .
bash .agents/skills/minight-terminal-mcp/scripts/sync-server.sh --check
```
