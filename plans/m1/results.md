# M1 — interactive launch: verification results

## Unit tests
- `internal/config` — 4 tests, all pass (Defaults, Load-missing,
  Load-overrides, Validate-rejects-unknown).
- `internal/detect` — 16 classifier cases still pass.
- `internal/harness` — 2 tests for {prompt} substitution (all occurrences,
  literals untouched, source not mutated).

## Subcommand smoke tests
- `wut harness list` — lists presets (aider, claude, codex) plus
  any user-defined harness; `*` marks active.
- `wut harness use <name>` — writes the config via tempfile +
  rename; `list` reflects the change on next run.
- `wut harness test --prompt "..."` — exec's directly into the
  configured command; {prompt} substituted.
- `wut doctor` — reports config path, active harness, default
  mode, and whether the active harness binary is on PATH.

## E2E in a hooked zsh subshell (fake harness → /bin/echo)
- NL routes: `how do I rebase onto main` → `[fake-harness]: how do I rebase
  onto main`, rc=0.
- Typo falls through: `gti status` → `zsh: command not found: gti`, rc=127.
- Real command untouched: `ls go.mod` runs normally.
- Recursion guard: after installing the hook then wiping PATH, typing a
  natural-language line prints plain `command not found` (one line, not a
  loop). Proves the `command -v wut` guard works.

## Exit-code contract
Revised from M0's "0 = handled, 127 = passthrough" to the exec-safe:
**127 = passthrough, anything else = handled**. The snippet propagates the
child's exit code back to the shell.

## Known M2 items
- Headless mode stub returns an error and downgrades to interactive with a
  stderr note. Headless lands in M2.
- `harness test` uses exec semantics — calling it replaces the current
  wut process with the harness. Expected; matches real runtime.
- `doctor`'s "shell hook installed?" check is advisory only; we print the
  eval line rather than parse rc files.
