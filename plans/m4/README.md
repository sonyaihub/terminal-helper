# Plan: `m4` — polish

Goal: close out the quality-of-life gaps before distribution. Confirm mode,
passthrough tokens, `harness add`, and nicer error messages.

Scope (from spec §14 M4):
- `behavior.confirm` — Y/n prompt before launching the harness
- `behavior.passthrough` — never route these first tokens even if NL-shaped
- `wut harness add` — register a custom harness from the CLI
- Audit + improve error messages

| # | Step | Status |
|---|---|---|
| 01 | Confirm mode (Y/n before launch) | done |
| 02 | Passthrough tokens (config → classifier) | done |
| 03 | `harness add` subcommand | done |
| 04 | Error-message audit & polish | done |
| 05 | Tests + E2E | done |
