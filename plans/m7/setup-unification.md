# Plan: `m7` — unify onboarding behind `wut setup`

## Goal

Make `wut setup` the one command a new user runs to go from zero to a working
shell hook + completion. Today it only writes `config.toml`; users then have
to separately run `wut install-hook` and (eventually) `wut install-completion`.
Forgetting `install-hook` is the single most common "wut doesn't work for me"
support question.

Keep the standalone commands callable (marked `Hidden: true`) so the multi-
shell and scripting cases still have an escape hatch.

## User experience

### Before

```
$ wut setup
? Which harness do you use?            [picker]
? Default mode?                        [picker]
✔ wrote ~/.config/wut/config.toml (harness=claude, mode=interactive)
  run `wut doctor` to verify

$ wut install-hook                    # easy to forget
$ wut                                  # nothing happens; shell says "command not found"
```

### After

```
$ wut setup
? Which harness do you use?            [picker]
? Default mode?                        [picker]
? Install shell hook? (Y/n) ▌          [defaults to yes, required for routing]
? Install shell completion? (y/N) ▌    [defaults to no, optional polish]
✔ wrote ~/.config/wut/config.toml
✔ appended hook to ~/.zshrc
  open a new shell or run: source ~/.zshrc
```

Non-interactive flag shape is preserved:

```
wut setup --harness codex --mode headless           # existing flags
wut setup --harness codex --mode headless --no-hook --no-completion
wut setup --shell fish                              # override shell detection
```

## Design decisions

### Keep standalone commands, mark them `Hidden`

Removing `install-hook` outright is a breaking change for:
- the caveats in `Formula/wut.rb` and `.goreleaser.yaml` (we just added those in #14)
- any docs / blog posts / muscle memory that references `wut install-hook <shell>`
- the multi-shell case: user already ran setup on zsh, now wants `wut install-hook fish` without re-running the full wizard

Setting `Hidden: true` on the command hides it from `wut --help` output while
keeping it callable. Zero LOC cost, preserves the primitive, lets `setup`
delegate to it.

Same treatment for the new `install-completion` command.

### Completion prompt defaults to **no**

Installing the shell hook is a hard requirement for `wut` to do anything at
the shell prompt — defaulting to yes is correct. Completion is polish: it
writes another file, may modify `.zshrc` to add an `fpath=` line, and some
users (dotfile purists) specifically don't want us touching filesystem paths
they didn't ask for. Default-no keeps the footprint small; power users opt in.

### `install-completion` goes into `~/.zsh/completions` + `.zshrc` fpath line

The cleanest workaround for zsh-config-variance (oh-my-zsh vs vanilla vs
prezto vs custom fpath) is to bypass framework detection entirely: write the
completion script to a path we own (`~/.zsh/completions/_wut`) and ensure
`fpath=(~/.zsh/completions $fpath)` exists in `.zshrc`. Both steps idempotent.

This mirrors `install-hook`'s pattern (also appends to `.zshrc` once, guarded
by a marker string). Acceptable because:
- Only one new line in `.zshrc`, guarded by the existing marker-comment convention
- Path is a user-owned directory, no sudo
- Users who don't want us touching `.zshrc` can run with `--print` and do it themselves

For bash and fish, standard paths exist and work without touching rc files:
- bash → `$(brew --prefix 2>/dev/null)/etc/bash_completion.d/wut`, falling
  back to `~/.local/share/bash-completion/completions/wut`
- fish → `~/.config/fish/completions/wut.fish`

### Soft failure, not hard error

If `install-hook` or `install-completion` fails mid-setup (no `$SHELL`,
unwritable rc file, whatever), `setup` still prints the config success and
surfaces a warning — it does *not* bail. The user got *most* of what they
wanted and can fix the rest with the explicit commands.

## Implementation

### Step 1 — `cmd/wut/install_completion.go` (new)

Mirror `install_hook.go` structurally. The key function signatures:

```go
func NewInstallCompletionCmd() *cobra.Command
// Flags: --shell, --print, --yes

func installCompletion(sh string, yes, print bool) error
// Writes the completion script via cobra.GenZshCompletion etc. to the right
// path and (for zsh) ensures the fpath line is present in .zshrc.
```

Supporting helpers (package-private):

```go
func completionPath(sh string) (string, error)   // per-shell default path
func ensureZshFpath(rc, dir string) error        // idempotent fpath= line
```

Reuse `detectShell()` and `rcPath()` from `install_hook.go` — move them to a
shared file if that feels cleaner, or leave as-is (Go's package scope makes
the sharing free either way).

### Step 2 — extract shell-install internals into callable helpers

`install_hook.go` currently does everything inside `RunE`. Pull the real work
into a plain function so `setup` can call it without re-parsing cobra flags:

```go
func installHook(sh string, yes bool) error {
    switch sh {
    case "zsh": return installRcLine("zsh", rcPath("zsh"), `eval "$(wut init zsh)"`, yes)
    case "bash": return installRcLine("bash", rcPath("bash"), `eval "$(wut init bash)"`, yes)
    case "fish": return installFishConfD(yes)
    default: return fmt.Errorf("unsupported shell %q", sh)
    }
}
```

The existing `RunE` becomes a one-liner that calls `installHook(...)`. Same
refactor for `install-completion` once it exists.

### Step 3 — wire prompts into `cmd/wut/setup.go`

Add two new flags and two new post-config-write blocks:

```go
cmd.Flags().BoolVar(&hookFlag, "install-hook", true, "install shell hook (use --install-hook=false or --no-hook to skip)")
cmd.Flags().BoolVar(&completionFlag, "install-completion", false, "install shell completion")
cmd.Flags().StringVar(&shellFlag, "shell", "", "override shell detection (zsh|bash|fish)")
cmd.Flags().Bool("no-hook", false, "...")       // Cobra idiom for negating a default-true flag
cmd.Flags().Bool("no-completion", false, "...")
```

After `writeConfig(path, cfg)`:

```go
sh := shellFlag
if sh == "" { sh = detectShell() }

installHookNow := hookFlag && !cmd.Flags().Changed("no-hook")
if installHookNow && !nonInteractive {
    installHookNow, _ = ui.Confirm("install shell hook?")
}
if installHookNow {
    if err := installHook(sh, true); err != nil {
        fmt.Fprintf(os.Stderr, "⚠ hook install failed: %v\n", err)
        fmt.Fprintf(os.Stderr, "  run `wut install-hook` manually when ready\n")
    }
}

installComplNow := completionFlag
if !completionFlag && !nonInteractive {
    installComplNow, _ = ui.Confirm("install shell completion?")
}
if installComplNow {
    if err := installCompletion(sh, true, false); err != nil {
        fmt.Fprintf(os.Stderr, "⚠ completion install failed: %v\n", err)
    }
}
```

"Non-interactive" here means `--harness` and `--mode` both supplied (matches
existing setup logic on line 26). In non-interactive mode, flags alone decide
what runs — no prompts.

### Step 4 — mark the standalone commands `Hidden`

In `NewInstallHookCmd()` and `NewInstallCompletionCmd()`:

```go
cmd.Hidden = true
```

They still respond to `wut install-hook zsh` and `wut install-completion zsh`,
they just don't clutter `wut --help`.

### Step 5 — update caveats and docs

- `Formula/wut.rb` caveats: replace the three-line setup with `wut setup`
- `.goreleaser.yaml` `caveats` block: same
- `README.md` install sections: point at `wut setup` as canonical
- `wut doctor`: if hook missing, print "run `wut setup` (or `wut install-hook`)" instead of just the latter

### Step 6 — tests

CLI tests in `cmd/wut/cli_test.go`:

- `TestSetupNonInteractiveInstallsHook` — with `--harness/--mode/--install-hook`
  pointing at a temp `$HOME`, assert `.zshrc` gets the eval line
- `TestSetupNonInteractiveSkipsHookWhenFlagged` — `--no-hook` leaves rc untouched
- `TestSetupNonInteractiveInstallsCompletion` — `--install-completion` writes
  `~/.zsh/completions/_wut` and the fpath line
- `TestInstallCompletionIdempotent` — running twice is a no-op second time
- `TestInstallCompletionPrint` — `--print` doesn't touch the filesystem
- `TestInstallHookStillCallable` — confirms the command is hidden but works
  (regression guard against accidental full removal)

The interactive-prompt path can't be driven from tests cleanly (needs a TTY),
but that's already true of the existing setup picker — same limitation, same
approach (cover the non-interactive path thoroughly).

## Files touched

| File | Change |
|------|--------|
| `cmd/wut/install_completion.go` | **new** — command + helpers |
| `cmd/wut/install_hook.go` | refactor `RunE` to call `installHook()`; add `Hidden: true` |
| `cmd/wut/setup.go` | add shell/hook/completion flags + post-write prompts |
| `cmd/wut/main.go` | register `NewInstallCompletionCmd()` |
| `cmd/wut/cli_test.go` | new tests + wire new cmd into `runCLI` |
| `cmd/wut/doctor.go` | update error messages pointing at `wut setup` |
| `Formula/wut.rb` | update caveats |
| `.goreleaser.yaml` | update `brews.caveats` |
| `README.md` | simplify install sections |

## Non-goals

- Detecting and supporting oh-my-zsh / prezto / etc. specifically. We write to
  a user-owned path + add one line to `.zshrc`; frameworks that override `fpath`
  may need a manual step (documented in caveats if it turns out to matter).
- Touching system-wide completion directories (`/etc/bash_completion.d`,
  `/usr/local/share/zsh/site-functions`). Too invasive, requires sudo.
- Dynamic-value completion (harness names, keyword placements as completion
  candidates). Still deferred per `plans/m6/completion.md` non-goals.
- Removing `install-hook` / `install-completion` outright. Marked `Hidden`
  instead; a future major version can drop them if telemetry shows nobody
  calls them.

## Estimated size

- `install_completion.go` ≈ 130 LOC (mirrors `install_hook.go` size)
- `setup.go` additions ≈ 60 LOC
- Refactor of `install_hook.go` ≈ 20 LOC delta
- Tests ≈ 150 LOC
- Docs/caveats ≈ 20 LOC delta

One PR is the right shape — the pieces are interdependent (setup calling
functions that live in install_*.go, hidden flags on commands that setup now
covers). Splitting introduces weird intermediate states where either setup is
broken or the new flow isn't reachable.
