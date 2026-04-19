# Plan: `wut why` — classifier introspection

## Goal

Given a shell input line, print what the classifier *would* do and which
signals drove the decision — without invoking the harness.

Pairs with the recently-shipped `wut keywords` command: after a user adds a
keyword, they currently have no way to verify it actually makes lines route
short of typing at a live prompt. `wut why` closes that loop.

## User experience

```
$ wut why "deploy the service pls"
"deploy the service pls"
  → would ROUTE to harness
  first token: deploy
  tokens: 4
  signals (2 of 2 required):
    ✔ stopword hit: "the" (built-in)
    ✔ first token is interrogative: "deploy" (user-added)

$ wut why "ls -la /etc"
"ls -la /etc"
  → would PASSTHROUGH
  hard gate tripped: first token contains path character

$ wut why "? rebase this branch"
"? rebase this branch"
  → would ROUTE to harness
  prefix override: "?" (forces route, strips prefix)
  stripped line: "rebase this branch"
```

Non-goals:

- No harness invocation — strictly offline.
- No JSON output — text only in v1.
- No batch mode (reading multiple lines from stdin). Could be added later if
  anyone asks.

## Implementation

### Step 1 — `internal/detect.Explain`

Today `detect.Classify` returns only `Classification` and `detect.Parse`
returns `Result{Class, Forced, Line}`. Neither carries why the decision was
made. Add a new function alongside them:

```go
// in internal/detect/heuristic.go

type Explanation struct {
    Class           Classification
    Forced          ForcedMode
    Line            string   // post-prefix-strip, for display
    FirstToken      string
    TokenCount      int
    HardGate        string   // non-empty when a hard gate short-circuited
    PrefixOverride  string   // non-empty when a prefix forced the decision
    Signals         []Signal // soft signals that fired (empty when hard-gated)
}

type Signal struct {
    Name   string // e.g. "stopword hit", "first token is interrogative"
    Token  string // the matching token
    Source string // "built-in" | "user-added"
}

func Explain(line string, opts ...Options) Explanation
```

Keep `Classify` / `Parse` unchanged — they're used in the hot path and on the
shell-hook codepath where allocation matters. `Explain` builds an
`Explanation` as it replays the same checks and is only called from the
human-facing `wut why` command.

Implementation mirrors `Classify` line-for-line but, instead of early-returning
on each check, records into the `Explanation` and keeps going where needed
(e.g. signal enumeration should report *all* fired signals, not stop at 2).
Hard gates (empty line, `<3` tokens, path chars, shell metacharacters) still
short-circuit — they set `HardGate` and skip soft-signal enumeration.

Prefix handling (`??`, `?!`, `?`, `\`, `!`) runs first and sets
`PrefixOverride`. When a prefix fires, the soft-signal list is empty because
the prefix alone is decisive.

### Step 2 — CLI command

New file `cmd/wut/why.go`:

```go
func NewWhyCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "why <line>",
        Short: "Explain how the classifier would handle a line (without invoking the harness).",
        Args:  cobra.MinimumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            cfg, _, err := loadConfig()
            if err != nil {
                return err
            }
            line := strings.Join(args, " ")
            exp := detect.Explain(line, detect.Options{
                Passthrough:         cfg.Behavior.Passthrough,
                ExtraStopwords:      cfg.Detection.ExtraStopwords,
                ExtraInterrogatives: cfg.Detection.ExtraInterrogatives,
            })
            printExplanation(line, exp)
            return nil
        },
    }
}
```

`printExplanation` is a private helper in the same file. Keep it dumb:
plain `fmt.Printf` against the shape shown in "User experience" above.

Register in `cmd/wut/main.go` next to the other commands.

### Step 3 — Tests

Unit (`internal/detect/heuristic_test.go`):

- `Explain("")` → `HardGate="empty line"`, empty signals.
- `Explain("ls")` → `HardGate="fewer than 3 tokens"`.
- `Explain("ls -la /etc")` → `HardGate` mentions path character.
- `Explain("how do I rebase")` → `Class=Route`, 2+ signals including the
  interrogative hit.
- `Explain("? anything works")` → `PrefixOverride="?"`, `Class=Route`.
- `Explain("deploy the thing", Options{ExtraInterrogatives: []string{"deploy"}})`
  → reports the signal with `Source="user-added"`.

CLI (`cmd/wut/cli_test.go`):

- Run `wut keywords add deploy --first-word`, then `wut why "deploy the service pls"`,
  assert output contains `ROUTE` and the added keyword.
- Run `wut why "ls -la /etc"` on a fresh config, assert output contains
  `PASSTHROUGH` and mentions the hard gate.

## Files touched

- `internal/detect/heuristic.go` — new `Explain` + types
- `internal/detect/heuristic_test.go` — new cases
- `cmd/wut/why.go` — new file
- `cmd/wut/main.go` — register `NewWhyCmd()`
- `cmd/wut/cli_test.go` — integration tests + wire into `runCLI` root

## Non-goals

- JSON / machine-readable output.
- Batch mode from stdin.
- Coloured output. (Can layer on later; the text format is designed to be
  grep-friendly.)

## Estimated size

~150 LOC for `Explain` + types, ~50 LOC for the CLI command + printer, ~80
LOC for tests. One PR.
