# Plan: `wut harness remove` — close the CRUD gap

## Goal

The `harness` subcommand has `add`, `list`, `use`, and `test` — but no
`remove`. Today, deleting a configured harness means hand-editing
`~/.config/wut/config.toml`. Close the gap.

## User experience

```
$ wut harness list
* claude        modes: [interactive headless]
  aider         modes: [interactive]
  codex         modes: [interactive headless]
  my-wrapper    modes: [interactive]

$ wut harness remove my-wrapper
✔ removed harness "my-wrapper"

$ wut harness remove claude
Error: "claude" is the active harness — run `wut harness use <other>` first,
       or pass --force to remove and fall back to the next available harness.

$ wut harness remove ghost
Error: no harness named "ghost" — run `wut harness list` to see configured harnesses.
```

## Design decisions

- **Can't remove the active harness without `--force`.** Removing the active
  harness puts the config into an invalid state (`config.Validate` requires
  `active_harness` to have a matching `[harness.<name>]` block). Requiring an
  explicit switch first is the safer default.
- **`--force` does the right thing.** If `--force` is passed and the target
  *is* active, pick the next harness name alphabetically and set it active
  before removing. If there are zero other harnesses left, error — a config
  with no harnesses is unusable.
- **Removing a preset (claude/aider/codex) is allowed.** Presets are only
  injected by `config.Defaults()` when the file doesn't exist yet; once the
  file is written, a removed preset stays gone. This matches user intent — if
  you delete claude, you mean it.

## Implementation

### Step 1 — CLI command

Add to `cmd/wut/harness.go`:

```go
func newHarnessRemove() *cobra.Command {
    var force bool
    cmd := &cobra.Command{
        Use:   "remove <name>",
        Short: "Remove a harness from the config.",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            name := args[0]
            cfg, path, err := loadConfig()
            if err != nil {
                return err
            }
            if _, ok := cfg.Harness[name]; !ok {
                return fmt.Errorf("no harness named %q — run `wut harness list` to see configured harnesses", name)
            }
            if cfg.ActiveHarness == name {
                if !force {
                    return fmt.Errorf("%q is the active harness — run `wut harness use <other>` first, or pass --force", name)
                }
                next, err := nextHarnessAfterRemoval(cfg.Harness, name)
                if err != nil {
                    return err
                }
                cfg.ActiveHarness = next
                fmt.Printf("→ active_harness switched to %q\n", next)
            }
            delete(cfg.Harness, name)
            if err := writeConfig(path, cfg); err != nil {
                return err
            }
            fmt.Printf("✔ removed harness %q\n", name)
            return nil
        },
    }
    cmd.Flags().BoolVar(&force, "force", false, "allow removing the active harness (falls back to the next available harness)")
    return cmd
}

func nextHarnessAfterRemoval(harnesses map[string]config.Harness, removing string) (string, error) {
    names := make([]string, 0, len(harnesses))
    for n := range harnesses {
        if n != removing {
            names = append(names, n)
        }
    }
    if len(names) == 0 {
        return "", fmt.Errorf("refusing to remove the only remaining harness — add another one first with `wut harness add`")
    }
    sort.Strings(names)
    return names[0], nil
}
```

Register in `NewHarnessCmd()`:

```go
cmd.AddCommand(newHarnessList(), newHarnessUse(), newHarnessTest(), newHarnessAdd(), newHarnessRemove())
```

### Step 2 — Tests

Add to `cmd/wut/cli_test.go`:

- `TestHarnessRemoveNonActive`: add "custom" via CLI, remove it, assert it's
  gone from `harness list` output and from the TOML file.
- `TestHarnessRemoveActiveRequiresForce`: remove the active preset (claude)
  without `--force`, expect an error mentioning `--force`.
- `TestHarnessRemoveActiveWithForceSwitchesToNext`: with `--force`, expect
  `active_harness` to move to the next alphabetical harness.
- `TestHarnessRemoveLastHarnessRefused`: add one harness, switch to it,
  remove all others, then try to remove it with `--force`; expect the
  "only remaining harness" error.
- `TestHarnessRemoveUnknownErrors`: removing a nonexistent name errors out.

## Files touched

- `cmd/wut/harness.go` — one new command + one helper
- `cmd/wut/cli_test.go` — 5 new test functions

## Non-goals

- Prompting for confirmation on removal. The command is scriptable; users
  who want safety should rely on `--force` semantics.
- Bulk remove (`wut harness remove a b c`). YAGNI — ask for it if it matters.

## Estimated size

~50 LOC for the command + helper, ~100 LOC across 5 tests. One small PR.
