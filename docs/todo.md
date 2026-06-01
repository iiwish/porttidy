# Porttidy TODO

> Source of truth: [spec.md](spec.md)  
> Current priority: make automated cleanup trustworthy, not broad.

## v0.1 - Trustworthy CLI Cleanup

### P0 - Safety Contract

- [x] Replace broad dev signatures with specific dev-server signatures.
- [x] Prevent Codex app server and `node_repl` from being classified as cleanup candidates.
- [x] Add process safety levels: `safe_to_cleanup`, `needs_confirmation`, `blocked`.
- [x] Add reason fields for matching, orphan status, and blocking.
- [x] Make force cleanup target only `safe_to_cleanup` processes.
- [x] Make `kill --all` an expert path, not the default product path.
- [x] Add killer-level pre-signal checks for self/parent, system process flags, and safe cleanup fields.
- [x] Re-read process cwd before signaling and verify it is still inside target dirs.
- [x] Separate hard guardrails from user cleanup policy with `cleanup_decision`.
- [x] Add `user_signatures` and `ignore_dirs` policy controls.

### P1 - Classification Quality

- [x] Use path-aware target directory containment instead of raw string prefix matching.
- [x] Treat broad runtimes like `node`, `python`, `pnpm`, `yarn`, `bun`, and `deno` as context, not standalone safe matches.
- [x] Prefer specific commands such as `vite`, `next dev`, `astro dev`, `python -m http.server`, and `uvicorn`.
- [x] Add classification tests for dev servers, Codex helpers, editor helpers, and generic scripts.
- [x] Add an integration fixture for orphan process detection.

### P2 - CLI and Output

- [x] Decide whether to add `cleanup` as the user-facing command and keep `kill` as a lower-level alias.
- [x] Update TTY output to show safety level and reason in a compact form.
- [x] Update JSON output with stable fields for agent/session hooks.
- [x] Add exit code `2` for "candidates found, no cleanup performed" if it improves automation.
- [x] Keep `--dry-run` as the recommended first-run path.

### P3 - Release Hygiene

- [x] Remove generated binary from source control if this repo will be public.
- [x] Remove `.DS_Store` from source control.
- [x] Add focused tests before v0.1 release.
- [x] Add install instructions only after the safety contract is implemented.
- [ ] Prepare Homebrew only after repeat usage validates the CLI.

## Deferred

- [ ] Linux support.
- [ ] MCP server.
- [ ] Shell completion.
- [ ] Project-level `porttidy.toml`.
- [ ] `porttidy start` / `porttidy stop`.
- [ ] Watch mode or daemon.
- [ ] Desktop GUI.
- [ ] Windows support.
- [ ] Docker cleanup.

## Validation Checklist

Before calling v0.1 ready:

- [ ] `porttidy scan` does not list Codex helper processes as dev cleanup candidates.
- [x] `porttidy kill --orphan --force` cannot kill editor, terminal, browser, Codex, Docker, or system helpers.
- [ ] Orphan dev servers are detected through a Go-based integration fixture.
- [ ] Non-orphan dev servers are visible but not force-cleanup targets by default.
- [x] JSON output explains why each process was matched or blocked.
- [ ] `go test ./...` passes.
