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
        "MAX_TIMEOUT_SECONDS": "300"
      }
    }
  }
}
```

You can also point Cursor at the built binary inside this repo after `go build`.

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
  "verbose": false
}
```

Compact response:

```json
{
  "stdout": "ok\n",
  "stderr": "",
  "return_code": 0,
  "timed_out": false,
  "current_cwd": "/home/user/project",
  "truncated": false
}
```

With `verbose: true`, the response also includes `duration_ms`, `stdout_omitted_bytes`, `stderr_omitted_bytes`, `session_id`, and `env_changed_count`.

### `get_session`

Return the current working directory for a session.

```json
{
  "session_id": "default",
  "current_cwd": "/home/user"
}
```

Environment values are intentionally not returned to reduce token usage and avoid leaking secrets.

### `kill_session`

Reset a session back to its initial state.

```json
{
  "session_id": "default",
  "reset": true
}
```

## Behavior

- Missing `session_id` uses `default`.
- Unknown sessions are auto-created at the user's home directory.
- Each command runs in a short-lived shell using `$SHELL -lc`.
- After each command, the server captures final `cwd` and environment via an internal trailer and updates the session, including after non-zero exits.
- On timeout, the process group is killed and session state is not updated.
- Output is ANSI-stripped and truncated using head/tail retention when it exceeds the configured limit.
- Dangerous commands are rejected before execution with a short JSON error.

## Security Model

`minight-terminal` is a trusted local tool with best-effort guardrails. It is not a sandbox.

The server rejects obviously dangerous commands such as `rm -rf /`, fork bombs, filesystem formatting, destructive `dd`, and shutdown/reboot commands. This protection is heuristic and can be bypassed. Do not expose this server to untrusted users or remote networks.

## Limitations

- No persistent PTY or interactive terminal emulation.
- No streaming output.
- No background job management.
- Environment persistence is based on post-command trailer capture, not a long-lived shell process.
- Aliases and shell functions depend on shell startup files and may not match an interactive terminal exactly.

## Configuration

Environment variables:

- `MAX_TIMEOUT_SECONDS`: maximum allowed timeout per command. Default server max is 300 seconds unless overridden.

Shell selection uses `$SHELL`, falling back to `/bin/sh`.

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

## Roadmap

- Docker/sandbox execution mode
- Persistent shell / PTY mode
- Streaming output
- `list_sessions`
- Allowed cwd configuration
