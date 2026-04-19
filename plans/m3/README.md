# Plan: `m3` — setup UX

Goal: finish the "first 60 seconds" experience — a guided setup wizard, a
real `ask` mode, markdown rendering for headless output, and the two
per-prompt mode prefixes (`??` / `?!`).

Scope (from spec §14 M3 + §13 open q 1a):
- `wut setup` wizard (interactive + non-interactive forms)
- `ask` mode — prompt user which mode to use per invocation
- `markdown` render mode (buffered output → ANSI)
- Per-prompt prefixes: `??` forces headless, `?!` forces interactive

| # | Step | Status |
|---|---|---|
| 01 | Markdown renderer (buffered, ANSI) | done |
| 02 | `ask` mode — TUI picker with interactive/headless/cancel | done |
| 03 | `wut setup` wizard + non-interactive flags | done |
| 04 | Prompt-level prefixes (`??`, `?!`) in the classifier layer | done |
| 05 | Tests + E2E | done |

Non-goals:
- Confirm mode (`behavior.confirm`) — separate polish task, lands in M4.
- Mode promotion / continuation hotkeys — deferred per spec §13 1b.
