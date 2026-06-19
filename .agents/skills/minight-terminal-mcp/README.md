# minight-terminal-mcp Skill Template

Reusable Cursor Agent Skill for installing, verifying, and using the `minight-terminal` MCP server.

## Install via skills.sh

```bash
npx skills add bluev102/minight --skill minight-terminal-mcp -a cursor -y
bash .agents/skills/minight-terminal-mcp/scripts/install-project-mcp.sh
```

Then reload MCP in Cursor.

Expected listing after installs accumulate:
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

## Contents

- `SKILL.md`: agent instructions and trigger scenarios
- `reference.md`: MCP tool contract and behavior
- `examples.md`: workflows and MCP vs built-in comparison
- `server/`: bundled MCP server source for local build
- `scripts/sync-server.sh`: sync/check bundled server source (maintainers)
- `scripts/verify-minight-terminal.sh`: read-only setup diagnostics
- `scripts/install-project-mcp.sh`: build binary and register project MCP config

## What Gets Installed

The install script builds:

```text
.minight-terminal/bin/minight-terminal
```

and registers it in:

```text
.cursor/mcp.json
```

Project-level skill install is required. Global install (`-g`) is not supported by the installer.

## Maintainer Release Checklist

Before pushing a release:

```bash
bash .agents/skills/minight-terminal-mcp/scripts/sync-server.sh
bash .agents/skills/minight-terminal-mcp/scripts/sync-server.sh --check
go test ./...
bash .agents/skills/minight-terminal-mcp/scripts/verify-minight-terminal.sh
```

## Security

Scripts build and run a local MCP server on the host machine. Review scripts before running. The MCP server is a trusted local tool with best-effort guardrails, not a sandbox.

## Copy To Another Project Manually

```bash
cp -R .agents/skills/minight-terminal-mcp /path/to/other-project/.agents/skills/
bash /path/to/other-project/.agents/skills/minight-terminal-mcp/scripts/install-project-mcp.sh
```

## Notes

- Skill is designed for project scope (`.agents/skills/`).
- MCP config is project-local (`.cursor/mcp.json`).
- Bundled `server/` source is synced from repo root before publish.
