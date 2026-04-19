# 04 — `detect` stub

## Description
Implement `wut detect --line "<text>"` as a stub that logs what it received and exits 127. This lets us wire up the shell hook (step 05) and confirm the round-trip before the heuristic lands in step 06.

Exit-code contract (locks now, steps 06 and 07 build on it):
- `0` — handled; the hook should swallow the line and return 0.
- `127` — not handled; the hook should print the default `command not found` message and exit 127.

## Status
done

## Depends on
[02 — CLI framework](02-cli-framework.md)

## What / where to change

- `cmd/wut/detect.go` — new file.
  - `NewDetectCmd()` returning a `detect` Cobra command.
  - Flag: `--line` (string, required). `MarkFlagRequired("line")`.
  - `RunE`:
    1. Write `wut: detect received: <line>` to `os.Stderr` (so it's visible during hook testing but doesn't contaminate stdout).
    2. Return a `cobra.Command.SilenceErrors = true` equivalent setup that doesn't double-print.
    3. Exit with code 127 — Cobra doesn't exit 127 naturally, so return a sentinel error and handle it in `main.go`, OR call `os.Exit(127)` directly from the command (simpler for now; document the choice in a comment).
- `cmd/wut/main.go` — register `detect` on root.

Do not implement any classification yet. Every invocation exits 127.

## How to verify

```
./wut detect --line "hello world"; echo "exit=$?"
# stderr: wut: detect received: hello world
# stdout: (nothing)
# exit=127

./wut detect; echo $?
# Cobra reports the missing --line flag; non-zero (not 127)

./wut detect --line ""; echo $?
# empty line still reaches the stub and exits 127

./wut detect --help   # shows the --line flag
```
