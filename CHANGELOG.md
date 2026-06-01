# Changelog

## v0.1.0-alpha.1 - 2026-06-01

First public alpha release.

Porttidy is intentionally narrow in this release: a macOS-first CLI for policy-aware cleanup of abandoned local dev servers after AI coding sessions.

### Added

- `scan` command for discovering local dev-server processes in configured project directories.
- `cleanup` command as the recommended safe cleanup path, with `--dry-run` and `--force`.
- Expert `kill` command with stricter guardrails around broad cleanup.
- Hard safety guardrails for editors, browsers, terminals, Docker Desktop, system helpers, and coding-agent runtimes such as Codex, Claude Code, and opencode.
- User cleanup policy through `user_signatures` and `ignore_dirs`.
- Policy-level cleanup decisions: `auto_cleanup`, `ask_first`, `blocked`, and `ignored`.
- JSON output designed for scripts and AI-agent/session hooks.
- Exit code `2` when cleanup candidates are found but no cleanup was performed.
- Pre-signal validation before process termination, including current cwd containment checks.

### Notes

- macOS is the only supported platform for this alpha.
- Linux and Windows are explicitly not supported yet, even though the codebase can be cross-compiled.
- Homebrew distribution is deferred until repeat real-world usage validates the CLI shape.

### Validation

- `go test ./...`
- `go build -o /tmp/porttidy-build-check ./cmd/porttidy`
