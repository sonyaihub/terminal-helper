# Plan: `wut completion` — shell completion

## Goal

Generate shell completion scripts for zsh, bash, and fish so tab-completion
works on subcommands, flags, and (optionally) dynamic values like harness
names and keyword placements.

Cobra gives this to us almost for free via its built-in completion generators
(`GenZshCompletion`, `GenBashCompletion`, `GenFishCompletion`). Most of the
effort is packaging them behind a clean `wut completion <shell>` command and
documenting the install path per shell.

## User experience

```
$ wut completion zsh > "${fpath[1]}/_wut"
$ autoload -U compinit && compinit

$ wut <TAB>
completion   config       detect       doctor       harness
init         install-hook keywords     mode         run
setup        version

$ wut harness <TAB>
add       list      remove    test      use

$ wut keywords add --<TAB>
--anywhere    --first-word
```

Also: `wut completion --help` prints install instructions per shell so users
don't have to remember paths.

## Design decisions

- **One command, per-shell subcommand,** matching Cobra convention and
  `kubectl`/`gh`. Calling just `wut completion` prints help listing the three
  shells.
- **Emit to stdout only.** Never auto-install to `fpath`, brew prefix, or
  `~/.bashrc` — too many variations (brew, oh-my-zsh, manual, asdf…). Users
  pipe it themselves. The long-form help contains copy-pasteable snippets.
- **Dynamic completion is out of scope for v1.** Cobra supports
  `ValidArgsFunction` for things like "list configured harness names" after
  `wut harness use`, but wiring that up meaningfully requires touching every
  command. Ship static completion first, layer on dynamic values if anyone
  asks for it.

## Implementation

### Step 1 — CLI command

New file `cmd/wut/completion.go`:

```go
package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

const completionLongHelp = `Generate a shell completion script.

Install (zsh, oh-my-zsh-compatible):
  wut completion zsh > "${fpath[1]}/_wut"
  autoload -U compinit && compinit

Install (zsh, no framework):
  mkdir -p ~/.zsh/completions
  wut completion zsh > ~/.zsh/completions/_wut
  echo 'fpath=(~/.zsh/completions $fpath)' >> ~/.zshrc
  echo 'autoload -U compinit && compinit' >> ~/.zshrc

Install (bash, with bash-completion):
  wut completion bash > $(brew --prefix)/etc/bash_completion.d/wut

Install (fish):
  wut completion fish > ~/.config/fish/completions/wut.fish`

func NewCompletionCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:                   "completion <zsh|bash|fish>",
        Short:                 "Generate a shell completion script.",
        Long:                  completionLongHelp,
        Args:                  cobra.ExactArgs(1),
        DisableFlagsInUseLine: true,
        ValidArgs:             []string{"zsh", "bash", "fish"},
        RunE: func(cmd *cobra.Command, args []string) error {
            root := cmd.Root()
            switch args[0] {
            case "zsh":
                return root.GenZshCompletion(os.Stdout)
            case "bash":
                return root.GenBashCompletion(os.Stdout)
            case "fish":
                return root.GenFishCompletion(os.Stdout, true)
            }
            return fmt.Errorf("unsupported shell %q — supported: zsh, bash, fish", args[0])
        },
    }
    return cmd
}
```

Register in `cmd/wut/main.go`:

```go
root.AddCommand(NewCompletionCmd())
```

### Step 2 — Suppress the auto-generated `completion` sibling

Cobra auto-adds its own `completion` command when any command defines
completion. Since we're supplying our own, disable the default:

```go
// in cmd/wut/root.go, inside NewRootCmd()
cmd.CompletionOptions.DisableDefaultCmd = true
```

This prevents `wut` from showing both our `completion` and Cobra's
auto-generated one in help output.

### Step 3 — Tests

Add to `cmd/wut/cli_test.go`:

- `TestCompletionZshContainsFunction`: run `wut completion zsh`, assert
  output contains `_wut` (the generated completion function name).
- `TestCompletionBashContainsFunction`: same for bash, expects `_wut_` prefix.
- `TestCompletionFishContainsDirective`: same for fish, expects
  `complete -c wut`.
- `TestCompletionUnknownShellErrors`: `wut completion tcsh` errors out.

Don't snapshot the full output — Cobra updates its templates occasionally
and we don't want churn on every dependency bump.

## Files touched

- `cmd/wut/completion.go` — new file (~40 LOC)
- `cmd/wut/main.go` — one line to register
- `cmd/wut/root.go` — one line to disable the default completion cmd
- `cmd/wut/cli_test.go` — 4 small tests

## Non-goals

- Dynamic completion (harness names, keyword placements). Layer on later if
  users ask.
- Auto-install. Users pipe the script themselves.
- PowerShell. Cobra supports it, but the `wut` audience is Unix shells —
  skip until someone asks.

## Estimated size

~60 LOC total: 40 for the command, 20 for tests. One very small PR — a good
warm-up or end-of-day merge.
