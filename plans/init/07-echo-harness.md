# 07 — Echo harness

## Description
Replace the `[would launch harness with: ...]` placeholder with a real invocation path that calls a hardcoded echo "harness". This proves the launch wiring end-to-end without needing config (that lands in M1) and gives step 08 something meaningful to demo.

The echo harness is a dev-only stand-in: it prints `[echo-harness]: <prompt>` to stdout, waits zero time, and exits 0. Future steps swap the hardcoded harness for a config-driven one.

## Status
done

## Depends on
[06 — Detection heuristic](06-heuristic.md)

## What / where to change

- `internal/harness/echo.go` — new file.
  - `type EchoHarness struct{}`.
  - `func (EchoHarness) Run(ctx context.Context, prompt string) error` — writes `[echo-harness]: <prompt>\n` to `os.Stdout`, returns nil.
  - Intentionally *not* a generic interface yet — M1 will introduce `harness.Runner` properly once we have real invocation shapes to unify against.
- `cmd/wut/detect.go` — replace the stderr `[would launch...]` line with:
  ```go
  if err := (harness.EchoHarness{}).Run(cmd.Context(), line); err != nil {
      return err
  }
  return nil   // exit 0
  ```
  Still exit 0 on route, 127 on pass-through.

## How to verify

```
./wut detect --line "how do I rebase onto main"; echo $?
# stdout: [echo-harness]: how do I rebase onto main
# exit: 0

./wut detect --line "gti status"; echo $?
# no output
# exit: 127
```

In the hooked shell:
```
how do I rebase onto main
# [echo-harness]: how do I rebase onto main

what is a git stash
# [echo-harness]: what is a git stash

ls
# runs ls normally

gti status
# zsh: command not found: gti
```

If the hooked-shell outputs interleave weirdly (e.g. the prompt reprints before the harness output), that's probably stderr vs stdout buffering — echo harness writes to stdout which should be fine. Report the anomaly and we'll debug rather than patch around it.
