# M0 — walking skeleton: verification results

Ran the step 08 matrix in a zsh session with the snippet sourced via
`eval "$(./wut init zsh)"`.

## Real commands — must run normally
- [x] `ls` — runs normally, handler never fires.
- [x] `git status` — runs normally.
- [x] `cd ~` — changes directory, no handler output.
- [x] `echo hello world` — builtin, handler never fires.
- [x] `./script` (existing) — runs; zsh handles missing path itself.

## Typos / unknown commands — must fall through
- [x] `gti status` → `zsh: command not found: gti`, exit 127.
- [x] `sl` → `zsh: command not found: sl`.
- [x] `pythno script.py` → `zsh: command not found: pythno`.
- [x] `./nope` → zsh's own `no such file or directory`; handler does not fire.

## Natural language — must route
- [x] `how do I rebase onto main` → `[echo-harness]: how do I rebase onto main`.
- [x] `fizzbuzz is a common interview question` → routes.
- [x] `explain this regex in plain english` → routes.

## Escape hatch / passthrough prefix
- [x] `!...` → passes through (the classifier honors the `!` prefix; user must
      quote the `!` to defeat zsh history expansion, e.g. `'!gti' foo`).
- [ ] `? short` → **known zsh friction**: under default `NOMATCH`, zsh globs `?`
      as a 1-char filename pattern and errors before the handler fires. Works
      with `setopt no_nomatch` or `'?' short`. The classifier itself routes
      correctly — this is a shell-parsing caveat, not an M0 bug.
- [ ] `\ foo bar` → **known zsh friction**: zsh consumes `\ ` as an escaped
      space, so the handler receives ` foo bar` (2 tokens after trim) and
      passes through. Same shell-parsing caveat.

## Regression / edge checks
- [x] `foobar | grep x` — zsh parses the pipeline before dispatching; only
      `foobar` reaches the handler, which classifies as PassThrough (1 token).
      The pipe character never reaches `detect`, so the metachar gate is
      belt-and-suspenders. Matches spec §11.
- [x] Collisions with real commands named like NL words (e.g. macOS ships
      `/usr/bin/what`) — the handler never fires, so `what is ...` runs the
      real binary. Matches spec §5.

## Out-of-scope for M0
- `wut doctor` — deferred to M1 (see plan README). Not tested.
- Real harness config / `{prompt}` substitution — M1.
- `?` / `\` escape hatches need documentation of the zsh quoting quirks, or
  additional zsh-side handling in the snippet (e.g. `setopt no_nomatch` local
  to the handler). Deferred — the classifier's contract is correct; the
  wrapper can harden its shell-quoting story in M1.

## Done criteria
- Every row in the three primary tables behaves as specified.
- The two escape-hatch rows are flagged as zsh-quoting friction, not classifier
  bugs, and carried into M1.
