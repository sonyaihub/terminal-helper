# 06 — Detection heuristic

## Description
Replace the stub's always-127 behavior with the classifier from `spec.md` §6: hard gates + soft signals + escape/passthrough prefixes. Keep the heuristic in its own package so it's unit-testable and future steps (real harness launch, headless mode) can consume `Classify` without importing CLI glue.

## Status
done

## Depends on
[04 — `detect` stub](04-detect-stub.md)

## What / where to change

- `internal/detect/heuristic.go` — new file.
  - Type `Classification int` with values `PassThrough`, `Route`.
  - Exported `Classify(line string) Classification`.
  - Implement per spec §6 in this order:
    1. Trim leading/trailing whitespace.
    2. **Explicit passthrough:** first char `!` → `PassThrough`.
    3. **Explicit route:** first char `?` or `\` → `Route`.
    4. Tokenize on whitespace.
    5. **Hard gates (any fail → `PassThrough`):**
       - `len(tokens) >= 3`
       - first token contains none of `/ . - ~ $`
       - no unquoted shell metachars anywhere: `| > < & ; $( `` (straight-quote scan is fine for v1)
    6. **Soft signals (need ≥ 2 → `Route`, else `PassThrough`):**
       - any stopword from a hardcoded set (`the a is how what why can i my to do does should`)
       - contains `?`
       - contains `'` or `,`
       - `len(tokens) >= 6`
       - first token ∈ interrogatives (`how what why explain write make fix help`)
- `internal/detect/heuristic_test.go` — table-driven tests covering at minimum:
  - Real commands that must pass through: `ls -la`, `git status`, `cd ~/tmp`, `./script.sh`, `python3 -V`.
  - Typos that must pass through: `gti`, `sl`, `pythno script.py`, `gti statsu`.
  - Natural language that must route: `how do I rebase onto main`, `what is the difference between git reset and git revert`, `explain what this regex does in plain english`.
  - Escape hatch: `? one`, `\ foo bar`.
  - Passthrough prefix: `!how do I rebase onto main`.
  - Metachar present: `how do i grep | sort` → passthrough.
- `cmd/wut/detect.go` — update:
  - Call `detect.Classify(line)`.
  - On `Route`: print `[would launch harness with: <line>]` to stderr, exit 0.
  - On `PassThrough`: exit 127 (no stderr noise).
  - Remove the "detect received" log line — was just for step 05 wiring.

## How to verify

```
go test ./internal/detect/...                     # all table cases pass
go test ./internal/detect/... -run TestClassify -v  # readable output

./wut detect --line "how do I rebase onto main"; echo $?
# stderr: [would launch harness with: how do I rebase onto main]
# exit: 0

./wut detect --line "gti status"; echo $?
# no stderr
# exit: 127

./wut detect --line "? short"; echo $?
# routes despite being short (escape hatch)
# exit: 0

./wut detect --line "!how do I rebase"; echo $?
# passes through despite looking like NL
# exit: 127
```

In the hooked shell from step 05:
```
gti status
# -> command not found (classifier said PassThrough)

how do I rebase onto main
# -> [would launch harness with: how do I rebase onto main]
# -> no "command not found" (we handled it, exit 0)
```
