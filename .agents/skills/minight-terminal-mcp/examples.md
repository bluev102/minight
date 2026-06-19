# Minight Terminal MCP Examples

## Example 1: Basic Command

Request:
```json
{
  "command": "pwd",
  "session_id": "demo",
  "cwd": "/home/user/project"
}
```

Expected compact response:
```json
{
  "stdout": "/home/user/project",
  "stderr": "",
  "return_code": 0,
  "timed_out": false,
  "current_cwd": "/home/user/project",
  "truncated": false
}
```

## Example 2: Persistent cwd

Step 1:
```json
{"command":"cd /tmp && pwd","session_id":"demo"}
```

Step 2:
```json
{"command":"pwd","session_id":"demo"}
```

Expected: second command still reports `/tmp`.

Built-in terminal note: separate shell invocations may not keep cwd unless chained in one command.

## Example 3: Persistent env for toolchain

Step 1:
```json
{"command":"export PATH=/home/user/.tools/go/bin:$PATH","session_id":"build"}
```

Step 2:
```json
{
  "command":"go test ./internal/config -count=1",
  "session_id":"build",
  "cwd":"/home/user/project",
  "timeout":60
}
```

Use this when MCP PATH is missing tools available in built-in terminal.

## Example 4: Timeout handling

Request:
```json
{
  "command":"sleep 60",
  "session_id":"demo",
  "timeout":1
}
```

Expected:
```json
{
  "return_code":124,
  "timed_out":true,
  "stderr":"\ncommand timed out"
}
```

Do not assume cwd/env changed after timeout.

## Example 5: Safety rejection

Request:
```json
{"command":"rm -rf /","session_id":"demo"}
```

Expected:
```json
{
  "return_code":1,
  "error":"command rejected by safety guardrail"
}
```

## Example 6: Debug truncation

Request:
```json
{
  "command":"go test ./...",
  "session_id":"build",
  "timeout":120,
  "verbose":true
}
```

Use when output is truncated and you need:
- `duration_ms`
- omitted byte counts
- env change count

## MCP vs Built-in Terminal Decision Guide

| Scenario | Use MCP | Use Built-in |
|---|---|---|
| Multi-step build in same cwd | Yes | No |
| Need `export PATH` once then run tools | Yes | No |
| Need full nvm/alias shell immediately | No | Yes |
| Need complete raw logs | No | Yes |
| Need structured exit/timeout metadata | Yes | No |
| Need token-efficient output | Yes | No |

## Agent Checklist

- [ ] Verify MCP server is registered (`verify-minight-terminal.sh`)
- [ ] Read tool descriptors before first call
- [ ] Pick stable `session_id`
- [ ] Set `cwd` on first command if needed
- [ ] Export PATH in session if command-not-found occurs
- [ ] Check `return_code`, `timed_out`, `truncated`
- [ ] Reset with `kill_session` when workflow ends
