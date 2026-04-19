# M4 — polish: verification results

## Unit tests
All prior tests still green. Added:
- `TestParsePassthroughTokens` — user-configured first-token passthrough.

## Confirm mode
- `behavior.confirm = true` shows `→ open <harness> with this prompt? [Y/n]`
  on stderr before launching.
- Accepts Y / y / Enter → proceed. n → exit 0 (handled, no command-not-found).
  Ctrl-C / Esc → same (cancelled).
- Uses `/dev/tty` directly so piped stdout doesn't break it.
- No tty → falls open (proceeds with a stderr note) rather than silently
  swallowing the line.

## Passthrough tokens
Piped from `behavior.passthrough` into `detect.Parse` via an `Options` struct.
- First-token exact match → PassThrough, independent of the heuristic.
- Other-token lines still classify normally.

Verified: `howto use this thing now` → exit 127 (passthrough hit);
`how do I use this thing now` → routes.

## `harness add`
Creates a new harness block without hand-editing TOML.

    wut harness add my-agent \
      --command my-agent --args "--tui,{prompt}" \
      --headless-command my-agent --headless-args "--once,{prompt}" \
      --headless-render markdown --use

- Refuses if the name already exists (with an actionable message).
- `--use` flips `active_harness` to the new name in the same write.
- Stdout shows both actions (`✔ added`, `✔ active_harness set to`).

## Error-message audit
Before → after, user-visible paths:
- `harness binary "X" not found on PATH: exec: …` →
  `harness binary "X" not found on PATH — install it, or run
  \`wut setup\` to pick a different harness (…)`
- `no harness named "X"` →
  `no harness named "X" — run \`wut harness list\` to see
  available harnesses`
- `harness "X" has no headless block and headless_fallback=error` →
  adds `either set default_mode=interactive, change the fallback, or
  add a headless block`
- `harness "X" has no interactive block` →
  spells out the TOML snippet to add.

## Precedence map (final)
When multiple mode signals are in play:

1. `--mode` CLI flag (explicit override)
2. Prompt prefix (`??` / `?!`)
3. `cfg.DefaultMode`

Passthrough tokens and `!` prefix short-circuit before mode selection.
Confirm fires after mode is resolved, before the runner is invoked.

## Deferred
- The confirm helper hasn't been integration-tested with interactive TTY
  input; compile+smoke only. A real tty-backed test harness would mean
  spawning a pty and exercising the runner, which is disproportionate
  overhead for a 40-line helper.
