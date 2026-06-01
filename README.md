# Porttidy

Policy-aware cleanup for AI-started local dev servers.

Porttidy is a small CLI for developers who use coding agents and often end up with abandoned local dev servers. It scans configured development directories, applies hard guardrails, then uses your cleanup policy to decide what can be auto-cleaned, what should be reviewed, and what should be ignored.

It is intentionally not a desktop app, process dashboard, or generic port killer.

## Why It Exists

`lsof`, `kill-port`, `fkill`, and Activity Monitor can find and kill processes. They do not reliably answer the question that matters after an AI coding session:

> Should this local dev server be auto-cleaned, reviewed, or ignored?

Porttidy exists to make that judgment conservative, explainable, and scriptable.

## Usage

The current CLI supports scanning and safe cleanup:

```bash
# Scan configured development directories
porttidy scan

# Show only orphan candidates
porttidy scan --orphan

# Preview cleanup of orphan candidates
porttidy cleanup --dry-run

# Clean up orphan candidates without confirmation
porttidy cleanup --force

# Machine-readable scan output
porttidy scan --json
```

`kill` remains available as an expert command. `kill --all` cannot be combined with `--force`; automation should use `cleanup --force`.

## Install From Source

```bash
go install github.com/iiwish/porttidy/cmd/porttidy@latest
```

For local development:

```bash
git clone https://github.com/iiwish/porttidy.git
cd porttidy
make install
```

## Product Direction

The v0.1 goal is narrow:

- macOS-first CLI.
- Detect abandoned local dev servers.
- Avoid false positives.
- Explain why each process was matched.
- Provide stable JSON for agent/session hooks.
- Make force cleanup safe enough for automation.

The primary command shape is:

```bash
porttidy scan
porttidy cleanup --dry-run
porttidy cleanup --force
```

## Non-Goals

Porttidy v0.1 will not include:

- Desktop GUI.
- Project start/stop orchestration.
- Generic port killing as the main workflow.
- Watch mode or daemon.
- Docker cleanup.
- Windows support.
- MCP server.

Those may be considered later only if they strengthen the core cleanup loop.

## Configuration

Config is created at:

```text
~/.config/porttidy/porttidy.yaml
```

Example:

```yaml
target_dirs:
  - ~/self
  - ~/daas

ignore_dirs:
  - ~/self/critical-demo

dev_signatures:
  - vite
  - next dev
  - astro dev
  - python -m http.server

user_signatures:
  - air
  - go run ./cmd/server

denylist:
  - Code
  - Cursor
  - Google Chrome
  - Terminal
  - iTerm
  - Warp
  - Docker Desktop
  - Codex
```

Built-in denylist entries are always preserved. User config can extend cleanup policy, but ordinary config cannot unlock protected apps, terminals, browsers, system helpers, or coding-agent runtimes.

## Development

```bash
make build
make dev
make test
```

## Exit Codes

| Code | Meaning |
| --- | --- |
| 0 | Command completed successfully |
| 1 | Command failed |
| 2 | Cleanup candidates were found, but no cleanup was performed |

## Roadmap

### v0.1

- Confirm the safety contract in daily use.
- Add install and source-build documentation.
- Keep macOS as the only supported platform.
- Avoid expanding scope before repeat usage.

### v0.2

- Stable JSON contract for agents.
- Shell/session hook examples.
- Homebrew distribution if repeat usage validates the tool.

### Later

- Linux support.
- MCP server.
- Project-level config.

See [docs/spec.md](docs/spec.md) for the product contract.
