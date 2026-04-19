# Plan: `m1` — interactive launch

Goal: replace the echo harness with a real config-driven `exec` into a user-
configured harness (claude / aider / codex / custom), with preset harnesses,
`{prompt}` substitution, and enough tooling (`doctor`, `harness test`) for
users to debug their config.

Scope: **interactive mode only**. Headless is M2.

| # | Step | Status |
|---|---|---|
| 01 | Harden zsh snippet (recursion guard + revised exit contract) | done |
| 02 | Config package — schema, TOML loader, defaults+presets | done |
| 03 | Harness package — Runner interface, {prompt} substitution, interactive exec | done |
| 04 | Wire `detect` to the real Runner | done |
| 05 | `harness` command tree — `list`, `use`, `test` | done |
| 06 | `doctor` command | done |
| 07 | Tests + build + E2E in live zsh | done |

## Exit contract change (important)

M0 used: `0 = handled`, `127 = pass-through`. That can't survive `exec` — the
harness's exit code overwrites detect's. We now use:

- `127` → pass through (zsh prints "command not found")
- any other code → handled (propagate through to the shell)

The snippet is updated accordingly. See `internal/shell/snippets/zsh.sh`.

## Recursion guard

If `wut` falls off `PATH` after the hook is installed, the handler
would infinite-loop. The new snippet short-circuits with a plain
"command not found" in that case.
