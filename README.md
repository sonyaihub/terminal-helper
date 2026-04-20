# wut

You open a terminal, start typing `how do I rebase onto main without losing my stash`, and the shell yells `command not found`. wut catches that moment and hands the typed text off to your configured AI harness (Claude Code, aider, codex, or a custom CLI) as a prompt.

Zero latency on normal commands — we only run when the shell has already decided your first token isn't a real command.

## Example

```sh
$ how do I rebase onto main without losing my stash
╭─ claude ──────────────────────────────────────────────────────╮
│ Stash your work first with `git stash`, then run              │
│ `git rebase origin/main`. After the rebase, `git stash pop`   │
│ to re-apply…                                                  │
╰───────────────────────────────────────────────────────────────╯
$
$ gti status
zsh: command not found: gti
$ ls -la
(runs ls normally)
```

## Install

**curl | sh** — no Go required. Downloads the prebuilt binary from the latest GitHub release, verifies the sha256, and drops it in `/usr/local/bin` (or `~/.local/bin` if you can't sudo):

```sh
curl -fsSL https://raw.githubusercontent.com/sonyabytes/wut/main/scripts/install.sh | sh
```

Pin a version with `TH_VERSION=v0.1.0`, or override the install dir with `TH_INSTALL_DIR=...`.

**Homebrew**:

```sh
brew install sonyabytes/tap/wut
```

**From source** (needs Go 1.26+):

```sh
go install github.com/sonyabytes/wut/cmd/wut@latest
```

## Setup

```sh
wut setup            # picks harness, writes config, wires the shell hook
```

Open a new shell, then:

```sh
wut doctor           # sanity-check the install
```

`wut setup` auto-detects your shell from `$SHELL`. Override with `--shell zsh|bash|fish`, skip the hook install with `--install-hook=false`, or add shell completion with `--install-completion`. Non-interactive invocation: `wut setup --harness codex --mode headless`.

Rather wire the hook by hand?

```sh
# zsh
echo 'eval "$(wut init zsh)"' >> ~/.zshrc

# bash (requires 4.0+; macOS stock bash is 3.2)
echo 'eval "$(wut init bash)"' >> ~/.bashrc

# fish
wut init fish > ~/.config/fish/conf.d/wut.fish
```

## What it does

- **Detects** natural language at the prompt with near-zero false positives on real commands (heuristic: token count + stopword + interrogative + punctuation signals, with explicit metacharacter and path gates).
- **Routes** the line to your configured harness in one of three modes:
  - `interactive` — hand the terminal over (`exec` into the harness).
  - `headless` — one-shot answer streamed back inline with a box or markdown renderer; shell stays in control.
  - `ask` — arrow-key picker chooses per invocation.
- **Escape hatches** — prefix a line with `?` or `\` to force-route, `!` to force-passthrough. `??` forces headless, `?!` forces interactive.
- **Passthrough allowlist** — list first-tokens in `behavior.passthrough` that should never route even if NL-shaped.

## Commands

| Command | Purpose |
|---|---|
| `wut setup` | Guided config wizard — picks harness/mode, wires the shell hook, optionally installs completion. Flags: `--harness`, `--mode`, `--shell`, `--install-hook=false`, `--install-completion`. |
| `wut init zsh\|bash\|fish` | Print the shell-hook snippet. Useful for hand-wiring or sourcing from framework-managed dotfiles. |
| `wut harness list\|use\|test\|add` | Manage harnesses. `use` takes `--command <bin>` to swap the binary of a preset (e.g. point claude at `claude-yolo`). `test` invokes directly without running detection. |
| `wut detect --line "<text>"` | Classify + act. Used by the shell hook; exit 127 = pass through. |
| `wut run --line "<text>" [--mode ...]` | Force a launch, skipping detection. |
| `wut mode set <mode>` | Shortcut for `config set default_mode <mode>`. |
| `wut config path\|edit\|get\|set` | Inspect or modify the config file. |
| `wut doctor` | Verify config, harness binary, and hook install. |
| `wut version` / `-v` / `--version` | Print version. |

### Using a wrapper binary

If you already have a wrapper like `claude-yolo` that calls
`claude --dangerously-skip-permissions` under the hood, swap just the binary
and keep the preset's args:

```sh
wut harness use claude --command claude-yolo
```

That flips active to `claude` and rewrites `command = "claude-yolo"` in both
the interactive and headless blocks (leaving `args` intact).

## Configuration

Lives at `~/.config/wut/config.toml` (or `$XDG_CONFIG_HOME/wut/config.toml`). Missing file is fine — presets for claude / aider / codex ship with the binary.

```toml
active_harness = "claude"
default_mode   = "headless"

[behavior]
confirm           = false
spinner           = true
passthrough       = ["gti", "sl"]
headless_fallback = "interactive"

[harness.claude]
interactive = { command = "claude", args = ["{prompt}"] }

[harness.claude.headless]
command = "claude"
args    = ["-p", "{prompt}"]
render  = "box"
```

Full schema is in [`spec.md`](./spec.md) §7.

## Design

- Detection runs at the shell's `command_not_found_handler` / `_handle` / `fish_command_not_found` hook. We never wrap normal command execution.
- The exit-code contract between the hook and the binary: **127 = pass through** (shell prints its usual "command not found"); **anything else = handled** (propagated as the child harness's exit code).
- Interactive mode uses `syscall.Exec` so the harness fully owns the tty.
- Headless mode spawns without a pty, streams stdout through a renderer, forwards stderr, and supports SIGINT forwarding (double-Ctrl-C escalates to SIGKILL) and `timeout_sec`.

## Development

```sh
go build ./cmd/wut
go test ./...
```

Design docs and milestone plans live in [`plans/`](./plans/). Start with [`spec.md`](./spec.md).

## Security

wut only passes the detected text as a string argument to a user-configured local binary. It never executes the detected text as a shell command, never makes network calls on its own, and has no telemetry.
