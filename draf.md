Build an MCP terminal server in Go for Cursor with these requirements:

## Technology
- Use `github.com/modelcontextprotocol/go-sdk` (official SDK)
- Transport: stdio
- Async runtime: goroutines + channels

## Tools to implement

1. `run_command(command, session_id, timeout, cwd)`
   - Execute via `$SHELL -lc` to load full shell environment (.zshrc, PATH, aliases)
   - Return stdout, stderr, return_code, current_cwd
   - Support timeout (default 30s, configurable per call)
   - Stateful sessions: keep cwd between commands

2. `get_session(session_id)`
   - Return current cwd of a session

3. `kill_session(session_id)`
   - Reset session to home directory

## Security
- Blacklist: rm -rf /, fork bomb, format commands
- Timeout to prevent hanging commands

## Output handling
- Strip ANSI escape codes before returning
- Truncate output to ~3000 chars with `...[truncated]...` if longer
- Always return valid JSON

## Files
- `main.go`
- `go.mod`
- `README.md` with install instructions and `~/.cursor/mcp.json` config

## Build steps
- Provide single binary output
- Show how to compile: `go build -o mcp-terminal .`
- Show MCP config to register binary in Cursor

Start with `main.go`, then `go.mod`, then README. Write each file fully.