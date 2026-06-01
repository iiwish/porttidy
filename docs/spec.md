# Porttidy SSOT

> Status: v0.1 product contract  
> Scope: macOS-first CLI for policy-aware cleanup of abandoned local dev servers  
> Last updated: 2026-06-01

## 1. Product Thesis

Porttidy helps AI-heavy developers clean up abandoned local dev servers after coding-agent sessions using hard safety guardrails plus user-controlled cleanup policy.

It is not a general process manager, not a desktop app, and not a prettier wrapper around `lsof`. Its reason to exist is narrower:

> Make automated cleanup trustworthy and policy-aware enough to run at the end of an AI coding session.

The product wins by being conservative, explainable, and easy to embed in scripts or agent hooks. It loses if it tries to become a full lifecycle manager before users trust its process classification.

## 2. Target User

Primary user:

- Developers who frequently use local AI coding agents such as Codex, Claude Code, Cursor, or similar tools.
- They often let agents start Vite, Next, Python static servers, preview servers, or local API servers.
- They work across multiple repositories and hit port conflicts or leftover processes often enough that manual cleanup becomes annoying.

Non-primary user:

- Developers who only occasionally need to kill one port.
- Teams looking for a full process supervisor.
- Users who want a GUI task manager.

For non-primary users, existing tools such as `lsof`, `kill-port`, `fkill`, Activity Monitor, or process supervisors are good enough.

## 3. Core Problem

The user problem is not "how do I kill a port?"

The real problem is:

> After an AI coding session, the user cannot quickly tell which local dev servers should be auto-cleaned, reviewed, or ignored.

Manual commands can find processes, but they do not answer the risk questions:

- Which project does this process belong to?
- Is it a development server or an unrelated app helper?
- Was it likely left behind by a completed session?
- Can it be killed automatically without disrupting current work?

Porttidy should replace that judgment burden, not just shorten a command.

## 4. Positioning

Porttidy is a policy-aware cleanup primitive for AI-native local development workflows.

It should be:

- CLI-first.
- macOS-first for v0.1.
- Automation-friendly.
- Conservative by default.
- Explainable in both TTY and JSON output.

It should not be:

- A desktop client.
- A generic port killer.
- A process dashboard.
- A project launcher.
- A long-running daemon.
- A full project lifecycle manager.

## 5. Non-Goals

v0.1 explicitly does not include:

- Desktop GUI.
- `porttidy start` or project service orchestration.
- Generic "kill anything on this port" behavior as the main path.
- Docker container cleanup.
- Windows support.
- Linux support unless macOS trust model is already validated.
- MCP server.
- Watch mode or background daemon.
- Automatic cleanup of processes that require human judgment.

These may become later capabilities only if they strengthen the core cleanup loop.

## 6. Product Principles

Safety beats coverage.

- False positives are worse than false negatives.
- Missing a process is acceptable in v0.1.
- Killing an unrelated process is a product failure.

Explanation beats magic.

- Each matched process should expose why it was matched.
- Each kill target should expose why it is eligible for auto cleanup.
- JSON output is part of the product contract, not an afterthought.

Automation beats UI.

- The primary integration surface is CLI plus machine-readable output.
- A GUI would increase distribution and maintenance cost without improving the core job.

Small beats broad.

- The MVP should do one narrow cleanup job well.
- Features that make Porttidy feel like a platform should be delayed.

## 7. Safety Model

Porttidy separates hard safety guardrails from user cleanup policy.

Hard guardrails answer: "Must this process never be killed by normal cleanup?"

User policy answers: "Given it is not hard-blocked, should this process be auto-cleaned, shown for review, or ignored?"

Every discovered process keeps a compatibility `safety_level` and also exposes a policy-level `cleanup_decision`.

| Level | Meaning | v0.1 behavior |
| --- | --- | --- |
| `safe_to_cleanup` | High confidence abandoned dev server | Eligible for automatic cleanup |
| `needs_confirmation` | Likely relevant, but not safe enough for automation | Show to user, never force-kill by default |
| `blocked` | System app, editor, current shell, current agent, or unknown risk | Never kill |

Cleanup decisions:

| Decision | Meaning | v0.1 behavior |
| --- | --- | --- |
| `auto_cleanup` | Matches cleanup policy and hard guardrails | Eligible for `cleanup --force` |
| `ask_first` | Looks relevant but lacks enough evidence or policy confidence | Show for review, never force-kill by default |
| `blocked` | Hard-blocked by system/app/runtime protection | Never clean |
| `ignored` | Excluded by user policy | Do not show as normal cleanup candidate |

v0.1 automatic cleanup must only target `cleanup_decision=auto_cleanup`.

### 7.1 `safe_to_cleanup`

A process can be `safe_to_cleanup` only when all conditions are true:

- CWD is inside a configured target directory.
- Command matches a specific dev-server signature.
- Process is not Porttidy itself or its parent shell.
- Process is not a known editor, terminal, browser, agent runtime, desktop app, or system helper.
- Process is orphaned, or is otherwise proven to belong to a completed agent session.
- Preferably has a listening local port.

For v0.1, orphan status should be the main automatic cleanup criterion. Agent-session ownership can come later.

### 7.2 `needs_confirmation`

A process should be `needs_confirmation` when it looks relevant but lacks enough safety evidence.

Examples:

- A dev server in a target directory that is not orphaned.
- A process with a dev-server command but no listening port.
- A process matched by a broad runtime signature such as Node, Bun, or Python without a specific dev-server command.

These processes may be shown in `scan`, but must not be killed by `--force` defaults.

### 7.3 `blocked`

A process is `blocked` if it is likely unrelated or too risky.

Examples:

- Codex app server or `node_repl`.
- VS Code, Cursor, Chrome, Terminal, iTerm, Warp, Docker Desktop, Linear, Slack, system UI processes.
- Root/system processes.
- Processes outside configured target directories.
- The current Porttidy process or its parent shell.

Blocked processes may appear in verbose diagnostics but should not appear as normal cleanup candidates.

### 7.4 `ignored`

A process is `ignored` when it is under an `ignore_dirs` policy entry. This is user preference, not a hard safety block.

Ignored processes should not appear as normal cleanup candidates and must not be force-cleaned.

## 8. Detection Rules

Detection should be specific enough to avoid accidental matches.

Good signatures:

- `vite`
- `next dev`
- `astro dev`
- `nuxt dev`
- `webpack serve`
- `python -m http.server`
- `python3 -m http.server`
- `uvicorn`
- `flask run`
- `python manage.py runserver`

Risky signatures that should not alone classify a process as safe:

- `node`
- `python`
- `pnpm`
- `yarn`
- `npm`
- `bun`
- `deno`
- `tsx`
- `ts-node`

These broad runtime names can help explain ancestry or context, but they should not be enough to make a process killable.

User policy can add project-specific signatures through `user_signatures`. These signatures extend cleanup policy but do not override hard blocks for editors, browsers, terminals, system helpers, or coding-agent runtimes.

## 9. CLI Contract

The v0.1 product direction is cleanup-first.

Preferred user-facing commands:

```bash
porttidy scan
porttidy cleanup --dry-run
porttidy cleanup --force
```

`kill` is an expert command. `kill --all` is not the primary path and must not support non-interactive force mode. Automation should use `cleanup --force`.

### 9.1 `scan`

Purpose: discover relevant local dev processes and explain their classification.

Expected default:

- Show likely dev-server processes in configured target directories.
- Clearly mark safety level.
- Clearly mark orphan status.
- Do not over-emphasize process count.

Useful flags:

- `--orphan`: show only orphan candidates.
- `--json`: machine-readable output.
- `--port`: filter by port.
- `--pid`: filter by PID.
- `--since`: filter by process age.

### 9.2 `cleanup`

Purpose: clean up only abandoned dev servers that pass hard guardrails and match the user's auto-cleanup policy.

Expected default:

- Target only `cleanup_decision=auto_cleanup`.
- Support `--dry-run` as the recommended first run.
- Support `--force` for automation hooks.
- Never force-kill `needs_confirmation` or `blocked` processes.

### 9.3 `kill`

Purpose: expert-level process termination for users who intentionally want more control.

Rules:

- `kill --all --force` is not allowed.
- `kill --all` requires strong interactive confirmation.
- `kill --orphan --force` should only act on force-safe cleanup candidates.
- The recommended automation path remains `cleanup --force`.

### 9.4 Exit Codes

Target contract:

| Code | Meaning |
| --- | --- |
| 0 | Command completed successfully |
| 1 | Command failed |
| 2 | Cleanup candidates found, no cleanup performed |
| 3 | Cleanup partially failed |

Exit code `2` is useful for agent hooks and CI-style diagnostics.

## 10. JSON Contract

JSON output must be stable enough for scripts and agents.

Each process should eventually include:

```json
{
  "pid": 12345,
  "ppid": 1,
  "name": "node",
  "cmdline": "node ./node_modules/.bin/vite --host 127.0.0.1 --port 5173",
  "cwd": "/Users/example/self/app",
  "ports": [5173],
  "is_orphan": true,
  "safety_level": "safe_to_cleanup",
  "cleanup_decision": "auto_cleanup",
  "can_force_cleanup": true,
  "match_reason": "matched specific dev-server signature: vite",
  "orphan_reason": "ppid=1",
  "blocked_reason": ""
}
```

The exact schema may evolve before v1.0, but v0.1 should avoid opaque booleans as the only explanation.

Empty collections should be encoded as arrays (`[]`), not `null`, so agent hooks can parse results without special null handling.

## 11. Kill Rules

Before terminating a process, Porttidy must re-check:

1. Process still exists.
2. PID is not Porttidy or its parent shell.
3. CWD is still inside target dirs.
4. Safety level is still `safe_to_cleanup`, unless user explicitly selected an interactive target.
5. Process is not denylisted or blocked.

Termination sequence:

1. Send SIGTERM.
2. Wait briefly.
3. If still alive, send SIGKILL.
4. Report status per PID.

Audit logging is useful later, but it is not more important than correct classification.

## 12. Configuration

Default config path:

```text
~/.config/porttidy/porttidy.yaml
```

v0.1 config:

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

The default config should be useful without requiring users to study it.

User-provided denylist entries extend the built-in safety denylist. They must not replace built-in protections for editors, browsers, terminals, system helpers, or coding-agent runtimes.

## 13. MVP Scope

v0.1 is successful when:

- macOS scan is fast enough for interactive use.
- Orphan local dev servers are detected.
- Known app/helper processes are not cleanup candidates.
- Dry-run output is clear enough for a user to trust.
- JSON output is usable by an agent/session hook.
- `--force` only acts on high-confidence cleanup candidates.

v0.1 is not successful merely because it finds many processes.

## 14. Validation Plan

Construct local fixtures or integration scenarios for:

- Vite server running normally.
- Vite server orphaned.
- Next dev server running normally.
- A Go-based test HTTP server orphaned by a helper launcher.
- Codex app server / `node_repl` in a target directory.
- Editor helper process in a target directory.
- Generic Node script that is not a dev server.

Acceptance criteria:

- Zero blocked processes appear as force-cleanup candidates.
- Orphan dev servers become cleanup candidates.
- Non-orphan dev servers require confirmation or are scan-only.
- JSON includes enough reason fields for agent decisions.
- `go test ./...` passes.

## 15. Roadmap

### v0.1 - Trustworthy CLI Cleanup

- Tighten dev-server detection.
- Add safety levels and reason fields.
- Make orphan cleanup the primary automated path.
- Add `cleanup` as the safe command.
- Keep macOS as the only supported platform.
- Add classification tests.
- Update README around cleanup-first positioning.

### v0.2 - Agent Integration

- Add stable JSON contract.
- Add shell/session hook examples.
- Consider MCP only if CLI contract is already trusted.
- Consider Homebrew distribution.

### v0.3 - Platform Expansion

- Add Linux support if v0.1 has repeat usage.
- Improve process ancestry/session ownership.
- Add project-level config only if users need narrower control.

### Explicitly Deferred

- Desktop GUI.
- Long-running daemon.
- Project start/stop orchestration.
- Windows support.
- Docker cleanup.

## 16. Current Alpha Boundary

The current alpha is releasable for macOS-first manual adoption, but its boundary should stay tight:

- Termination re-checks self/parent, system-process flags, `can_force_cleanup`, `safety_level`, `cleanup_decision`, and current cwd containment before signaling.
- `kill --all` is guarded, but it still exists as an expert escape hatch and should stay out of onboarding docs.
- The default signature list should be validated against more real AI-agent sessions before a stable release.
- Homebrew and shell completion remain deferred until repeat usage proves the CLI shape.

Do not expand scope until real usage shows whether the cleanup decision model is trusted.
