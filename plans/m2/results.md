# M2 — headless mode: verification results

## Unit tests
- `internal/config` — existing 4 pass with new fields.
- `internal/detect` — 16 classifier cases.
- `internal/harness` — 3 cases (all occurrences, literal untouched, new
  headless fields preserved through Substitute).
- `internal/render` — 3 box tests (header/body/footer, partial-line
  buffering, trailing-flush on Close).

## E2E (all run in a fresh shell with a fake harness config)

### Box rendering streams line by line
With a 3-line sleeping script and `render = "box"`, `harness test` produced:

    ╭─ fake ──────────────────────────────────────────────...╮
    │ thinking about: squash commits                         │
    │ here is a stashed answer line                          │
    │ and another one for good measure                       │
    ╰────────────────────────────────────────────────────────╯

### Default mode wiring
- `default_mode = "headless"` in config → `detect` routes NL lines through
  the headless runner by default.
- `--mode=interactive` on `detect` / `harness test` overrides.

### Fallback policy
- `behavior.headless_fallback = "interactive"` + aider (no headless block)
  → silently switches to interactive invocation.
- `"error"` → exits with "harness has no headless block".
- `"ask"` → currently downgrades to interactive with a stderr note (wizard
  lands in M3).

### Exit-code propagation
- Child `exit 42` after box render → `wut` exits 42 (not Cobra's
  generic 1). No "Error: exit status 42" noise.
- Shell hook sees 42 → non-127 → "handled" branch → no `command not found`.

### Timeout
- `timeout_sec = 1` on a script that sleeps 10s → child killed at ~1.2s
  via SIGTERM to the process group. Signaled exits surface as rc=255.
- Process group creation (`Setpgid: true`) was necessary so shell scripts'
  grandchildren (like `sleep`) die with the parent.

### Classifier regressions
- `gti status` still passthrough.
- `how do I rebase onto main` still routes.

## Bugs fixed during M2
- `Substitute()` was dropping `Stream`, `Render`, `TimeoutSec` when copying
  the invocation; added explicit copies + test to prevent regression.
- Timeout implementation had to signal the process group, not the leader,
  because `/bin/sh` doesn't forward SIGTERM to foreground children.

## Deferred to M3
- Markdown render mode (currently aliased to raw).
- True `ask` prompt (prints note and falls back to interactive).
- `wut setup` wizard.
