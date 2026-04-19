# M3 — setup UX: verification results

## Unit tests
- `internal/render/markdown_test.go` — 4 cases (heading, inline
  bold+code, bullets, fenced code).
- `internal/detect/heuristic_test.go` — adds 7 Parse cases for prefixes.
- Prior 16 classifier cases + 4 config cases + 3 substitute cases unchanged.

## Markdown renderer
Buffered-only; streaming invocations with `render = "markdown"` fall back to
raw automatically. ANSI output for:
- `# heading` → bold + underline
- `**bold**` / `__bold__` → bold
- `*italic*` / `_italic_` → italic (boundary-aware so `foo_bar_baz` isn't
  accidentally styled)
- `` `inline code` `` → inverse
- Triple-backtick fences → dim, indented
- `-`/`*`/`+` list markers → `• ` bullet

## `ask` mode
- `default_mode = "ask"` → shows an arrow-key picker with headless / interactive / cancel.
- Picker reads from `/dev/tty`, so piped stdout doesn't break it.
- Cancel returns cleanly (exit 0, no "command not found").
- When stdout is a non-tty, ask degrades to interactive with a stderr note.

## Prompt-level prefixes (§13 1a)
All resolved at the classifier (Parse) layer before mode selection, so the
`--mode` flag on the CLI still takes precedence when set explicitly.

| Input | Class | Forced mode | Line passed to harness |
|---|---|---|---|
| `??how do I rebase onto main` | Route | headless | `how do I rebase onto main` |
| `?!fix this regex for me` | Route | interactive | `fix this regex for me` |
| `? short` | Route | — | `short` |
| `\ foo bar` | Route | — | `foo bar` |
| `!anything` | PassThrough | — | (unchanged) |

## `wut setup`
- Non-interactive: `setup --harness <name> --mode <mode>` writes config
  directly. Validates both args against the parsed config.
- Interactive: arrow-key picker for harness (annotates each entry as
  "detected on PATH" / "not installed") → picker for default mode →
  writes config + suggests running `doctor`.
- Uses the same atomic tempfile rename the `harness use` command does.

## E2E (live binary, fake harness config)
All of the following succeeded on the first pass:
1. `setup --harness md --mode headless`
2. `harness test --mode headless` against a markdown-producing fake →
   ANSI-formatted output.
3. `detect --line "??..."` → box-rendered headless output.
4. `detect --line "?!..."` → interactive exec.
5. `detect --line "? short"` → prefix stripped, routes the 1-token line.
6. `detect --line "gti status"` → exit 127 (unchanged).

## Known caveats
- The arrow-key picker hasn't been regression-tested with tmux / older
  terminals that send different escape sequences for ↑/↓; it accepts
  `\x1b[A`/`\x1b[B` and vi-style `k`/`j` as a fallback.
- On zsh, the `??` / `?!` prefixes still collide with the `NOMATCH` glob
  setting (same caveat as plain `?`). Users need to quote or
  `setopt no_nomatch`.
