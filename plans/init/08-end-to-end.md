# 08 — End-to-end verification

## Description
No code. A structured walk-through that proves the walking skeleton meets M0's exit criteria and catches regressions before we build on top of it. If anything in this matrix fails, fix it before starting M1.

## Status
done

## Depends on
All prior steps (01–07).

## What / where to change
Nothing in source. Optional: capture the results in `plans/init/results.md` as a checkbox list so future-you can diff against it when M1 lands.

## How to verify

In a fresh zsh session with the snippet sourced (per step 05):

### Real commands — must run normally
| Input | Expected |
|---|---|
| `ls` | Normal `ls` output. |
| `git status` | Normal git output (or git's own error if not a repo). |
| `cd ~` | Changes directory. No handler output. |
| `echo hello world` | Prints `hello world`. Note: `echo` is a builtin, so the handler never fires. |
| `./some-script-that-exists` | Runs the script. |

### Typos / unknown commands that are NOT natural language — must fall through
| Input | Expected |
|---|---|
| `gti status` | `zsh: command not found: gti`. No echo-harness. |
| `sl` | `zsh: command not found: sl`. |
| `pythno script.py` | `zsh: command not found: pythno`. |
| `./nope` | zsh's "no such file or directory" (handler still fires but classifier passes through — the `.` and `/` trip hard gates). |

### Natural language — must route to echo harness
| Input | Expected |
|---|---|
| `how do I rebase onto main` | `[echo-harness]: how do I rebase onto main`. |
| `what is the difference between reset and revert` | `[echo-harness]: what is the difference between reset and revert`. |
| `explain this regex in plain english` | `[echo-harness]: ...`. |

### Escape hatch / passthrough prefix
| Input | Expected |
|---|---|
| `? short` | Routes to echo-harness despite only 2 tokens after `?`. |
| `\ foo bar` | Routes. |
| `!how do I rebase onto main` | Falls through — `zsh: command not found: !how...` (or whatever zsh does with `!` — history expansion caveat; document if weird). |

### Regression checks
- `time ls` — `ls` still timed normally; no perceptible delay vs a shell without the hook.
- `foobar | grep x` — shell metachar trips hard gate → pass through, no handler action on the whole pipeline.
- Pasting a multi-line block with code symbols on line 1 — first line passes through cleanly (classifier rejects on metachars / `-` / `/`).

### Exit check
Run `./wut doctor` — doesn't exist yet in M0. That's expected. Note it in `plans/init/results.md` as a known M1 item.

## Done criteria
Every row in the three top tables behaves exactly as described, with no workarounds. If any row is flaky (e.g. only fails in tmux, only fails on macOS Terminal vs iTerm), file it in `plans/init/results.md` and decide whether to fix in M0 or carry as a known issue into M1.
