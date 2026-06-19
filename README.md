# minight-terminal

Local MCP stdio server that lets Cursor agents run shell commands on the host machine with session-aware `cwd` and environment persistence.

## Install via skills.sh

Recommended for Cursor users:

```bash
npx skills add bluev102/minight --skill minight-terminal-mcp -a cursor -y
bash .agents/skills/minight-terminal-mcp/scripts/install-project-mcp.sh
```

Then reload MCP in Cursor.

After installs accumulate, the skill may appear at:
`https://skills.sh/bluev102/minight/minight-terminal-mcp`

## Recommended Cursor Rule

Enabling the MCP server makes the tools available to Cursor, but agents may still
choose Cursor's built-in terminal for ordinary shell commands. To make usage more
consistent, add this as a Cursor user rule or project rule:

```md
When running shell commands, prefer the minight-terminal MCP when the task
benefits from persistent cwd/env, structured JSON output, explicit timeouts, or
repeatable command sessions.

Use Cursor's built-in terminal for interactive commands, long-running dev
servers/watchers, raw log inspection, or commands that depend heavily on the
user's shell aliases, nvm, or inherited local environment.

Before using any MCP tool, read its tool descriptor/schema first.
```

## Quickstart (from source)

Build the binary:

```bash
go build -o minight-terminal .
```

Register it in Cursor MCP config (`~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "minight-terminal": {
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

You can also point Cursor at the built binary inside this repo after `go build`.

## Windows Setup

### Native Windows (recommended for repos on `E:\`, `C:\`, etc.)

Build on Windows and register the binary directly:

```json
{
  "mcpServers": {
    "minight-terminal": {
      "command": "C:\\path\\to\\minight-terminal.exe",
      "env": {
        "MINIGHT_BACKEND": "windows",
        "DEFAULT_TIMEOUT_SECONDS": "60",
        "MAX_TIMEOUT_SECONDS": "300"
      }
    }
  }
}
```

Native Windows backend uses PowerShell, avoids WSL `/mnt/<drive>` filesystem penalties, and accepts Windows paths directly.

### WSL bridge (legacy)

If you must run through WSL, use `/mnt/<drive>/...` paths (not `/e/...`). Enable path normalization (default on) or set `MINIGHT_NORMALIZE_WSL_PATHS=true`.

Expect slower git/filesystem operations on drvfs mounts. Verbose responses include `environment_warnings` when cwd is on `/mnt/<drive>/`.

## Tools

### `run_command`

Execute a shell command in a session.

Input:

```json
{
  "command": "go test ./...",
  "session_id": "default",
  "timeout": 30,
  "cwd": "/home/user/project",
  "verbose": false,
  "fail_on_any_error": false,
  "pipefail": false,
  "strip_crlf": true
}
```

- `session_id` defaults to `default`.
- `timeout` defaults to `DEFAULT_TIMEOUT_SECONDS` (or 30s). Values above `MAX_TIMEOUT_SECONDS` are capped.
- `return_code` is the shell exit code of the full command string.
- `had_failure` is true when any command segment failed or exit code was non-zero.
- Use `fail_on_any_error: true` (bash) to detect earlier failures in `;` chains while keeping shell-final `return_code`.
- Use `pipefail: true` (bash) so pipeline failures propagate.

Compact response:

```json
{
  "stdout": "ok\n",
  "stderr": "",
  "return_code": 0,
  "timed_out": false,
  "current_cwd": "/home/user/project",
  "truncated": false,
  "had_failure": false,
  "cwd_persisted": true
}
```

With `verbose: true`, the response also includes `duration_ms`, `stdout_omitted_bytes`, `stderr_omitted_bytes`, `stdout_total_bytes`, `stderr_total_bytes`, `session_id`, `env_changed_count`, `environment_warnings`, and `suggested_timeout` on timeout.

### `get_session`

Return session metadata for a session.

```json
{
  "session_id": "default",
  "current_cwd": "/home/user",
  "env_key_count": 42,
  "last_command": "go test ./...",
  "background_jobs": 0,
  "last_return_code": 0,
  "last_had_failure": false
}
```

Environment values are intentionally not returned to reduce token usage and avoid leaking secrets.

### `list_sessions`

List active sessions with safe metadata (no env values).

### `kill_session`

Reset a session back to its initial state.

```json
{
  "session_id": "default",
  "terminate_background_jobs": true
}
```

Response:

```json
{
  "session_id": "default",
  "reset": true,
  "background_jobs_killed": 1
}
```

## Behavior

- Missing `session_id` uses `default`.
- Unknown sessions are auto-created at the user's home directory.
- Each command runs in a short-lived shell via the selected backend (`posix` or `windows`).
- After each command, the server captures final `cwd` and environment via an internal trailer and updates the session, including after non-zero exits.
- `current_cwd` always reflects the shell-reported cwd from the trailer; `cwd_persisted` confirms session storage matched.
- On timeout, the process group is killed and session state is not updated.
- Output is ANSI-stripped, CRLF-normalized (by default), and truncated using head/tail retention when it exceeds the configured limit.
- Dangerous commands are rejected before execution with a short JSON error.
- Background jobs started with `&` are tracked when detectable; `kill_session` with `terminate_background_jobs: true` kills tracked PIDs.

## Security Model

`minight-terminal` is a trusted local tool with best-effort guardrails. It is not a sandbox.

The server rejects obviously dangerous commands such as `rm -rf /`, fork bombs, filesystem formatting, destructive `dd`, and shutdown/reboot commands. This protection is heuristic and can be bypassed. Do not expose this server to untrusted users or remote networks.

## Limitations

- No persistent PTY or interactive terminal emulation.
- No streaming output.
- Background job tracking is best-effort for short-lived shells; jobs started outside trailer capture may not be tracked.
- Environment persistence is based on post-command trailer capture, not a long-lived shell process.
- Aliases and shell functions depend on shell startup files and may not match an interactive terminal exactly.

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DEFAULT_TIMEOUT_SECONDS` | `30` | Default per-command timeout |
| `MAX_TIMEOUT_SECONDS` | `300` | Maximum allowed timeout per command |
| `MINIGHT_BACKEND` | `auto` | `auto`, `posix`, or `windows` |
| `MINIGHT_SHELL` | `$SHELL` / `/bin/sh` | Shell binary override |
| `MINIGHT_STRIP_CRLF` | `true` | Strip `\r` from output |
| `MINIGHT_NORMALIZE_WSL_PATHS` | `true` | Convert `/e/...` to `/mnt/e/...` under WSL |
| `MINIGHT_OUTPUT_LIMIT` | `3000` | Max stdout/stderr chars before truncation |

## Development

Run unit tests:

```bash
go test ./...
```

Run black-box MCP stdio tests:

```bash
go test ./e2e
```

Build release binary:

```bash
go build -o minight-terminal .
```

Sync skill bundled server after root changes:

```bash
bash .agents/skills/minight-terminal-mcp/scripts/sync-server.sh
bash .agents/skills/minight-terminal-mcp/scripts/sync-server.sh --check
```

## Roadmap

- Docker/sandbox execution mode
- Persistent shell / PTY mode
- Streaming output
- Allowed cwd configuration
