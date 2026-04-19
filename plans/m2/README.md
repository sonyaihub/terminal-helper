# Plan: `m2` — headless mode

Goal: add a real headless code path — spawn the harness as a child, stream
stdout through a renderer, forward stderr, support interrupts and timeouts,
and let users pick headless as their default mode.

Scope (from spec §14 M2): streaming stdout, `box` renderer, interrupt
handling, timeout, fallback policy, configurable default mode.

Deferred to M3: `markdown` render, `ask` mode prompt, `wut setup`
wizard.

| # | Step | Status |
|---|---|---|
| 01 | Config additions (spinner, fallback, per-mode render/stream/timeout) | done |
| 02 | `internal/render/box.go` — box renderer wrapping streamed lines | done |
| 03 | `internal/render/spinner.go` — dots spinner, erasable | done |
| 04 | `internal/harness/headless.go` — child, pipes, renderer, signals, timeout | done |
| 05 | Wire `detect` / `harness test` to honor `default_mode` / `--mode` / fallback | done |
| 06 | Tests + E2E | done |

Non-goals for M2:
- No pty allocation — spec §9 is explicit.
- No markdown rendering (lands in M3).
- No `ask` prompt (M3).
