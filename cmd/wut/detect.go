package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sonyaihub/wut/internal/config"
	"github.com/sonyaihub/wut/internal/detect"
	"github.com/sonyaihub/wut/internal/harness"
	"github.com/sonyaihub/wut/internal/ui"
)

func NewDetectCmd() *cobra.Command {
	var line string
	var modeFlag string
	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Classify a shell input line and, if natural language, route it to the active harness.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig()
			if err != nil {
				return err
			}
			res := detect.Parse(line, detect.Options{
				Passthrough:         cfg.Behavior.Passthrough,
				ExtraStopwords:      cfg.Detection.ExtraStopwords,
				ExtraInterrogatives: cfg.Detection.ExtraInterrogatives,
			})
			if res.Class != detect.Route {
				// Exit 127 to signal the shell hook that we did not handle
				// the line; zsh will then emit its usual "command not found".
				os.Exit(127)
				return nil
			}
			mode := cfg.DefaultMode
			if res.Forced != detect.ForceNone {
				mode = config.Mode(res.Forced)
			}
			if modeFlag != "" {
				mode = config.Mode(modeFlag) // explicit flag wins over prefix
			}
			return propagateExit(runHarness(cmd, cfg, cfg.ActiveHarness, mode, res.Line))
		},
	}
	cmd.Flags().StringVar(&line, "line", "", "the raw input line from the shell")
	cmd.Flags().StringVar(&modeFlag, "mode", "", "override default_mode (interactive|headless|ask)")
	_ = cmd.MarkFlagRequired("line")
	return cmd
}

// runHarness applies ask/confirm/fallback policy, picks the right runner,
// and dispatches. Shared by `detect`, `run`, and `harness test`.
func runHarness(cmd *cobra.Command, cfg *config.Config, name string, mode config.Mode, prompt string) error {
	h, ok := cfg.Harness[name]
	if !ok {
		return fmt.Errorf("no harness named %q — run `wut harness list` to see available harnesses", name)
	}

	// Ask mode — interactive picker. On non-tty or cancel, fall through to
	// interactive so the user still gets *some* response.
	if mode == config.ModeAsk {
		chosen, err := askForMode(name)
		if err != nil {
			if errors.Is(err, ui.ErrCancelled) {
				return nil // user chose cancel — treat as handled, exit 0
			}
			fmt.Fprintf(os.Stderr, "wut: ask mode unavailable (%v), using interactive\n", err)
			mode = config.ModeInteractive
		} else {
			mode = chosen
		}
	}

	// Headless fallback when the harness has no headless block.
	if mode == config.ModeHeadless && h.Headless == nil {
		switch cfg.Behavior.HeadlessFallback {
		case config.FallbackError:
			return fmt.Errorf("harness %q has no headless block and behavior.headless_fallback=error — either set default_mode=interactive, change the fallback, or add a headless block", name)
		case config.FallbackAsk:
			chosen, err := askForMode(name)
			if err != nil {
				if errors.Is(err, ui.ErrCancelled) {
					return nil
				}
				mode = config.ModeInteractive
			} else {
				mode = chosen
			}
		case config.FallbackInteractive, "":
			mode = config.ModeInteractive
		}
	}

	if cfg.Behavior.Confirm {
		ok, err := ui.Confirm(fmt.Sprintf("→ open %s with this prompt?", name))
		if err != nil {
			if errors.Is(err, ui.ErrCancelled) {
				return nil
			}
			// No tty — default to proceeding rather than surprising the user
			// with a silent no-op.
			fmt.Fprintf(os.Stderr, "wut: confirm unavailable (%v), proceeding\n", err)
		} else if !ok {
			return nil // user declined — handled, exit 0
		}
	}

	switch mode {
	case config.ModeInteractive:
		if h.Interactive == nil {
			return fmt.Errorf("harness %q has no [harness.%s.interactive] block — add one with command = \"<bin>\", args = [\"{prompt}\"]", name, name)
		}
		return harness.Interactive{Invocation: h.Interactive}.Run(cmd.Context(), prompt)
	case config.ModeHeadless:
		return harness.Headless{
			Invocation: h.Headless,
			Opts: harness.HeadlessOptions{
				Label:          name,
				SpinnerEnabled: cfg.Behavior.SpinnerEnabled(),
				SpinnerStyle:   spinnerStyleOrDefault(cfg.Behavior.SpinnerStyle),
			},
		}.Run(cmd.Context(), prompt)
	default:
		return fmt.Errorf("mode %q is not one of interactive|headless|ask — check --mode / default_mode / prompt prefix", mode)
	}
}

func spinnerStyleOrDefault(s string) string {
	if s == "" {
		return "dots"
	}
	return s
}

// askForMode presents a picker for "how should I open <harness>?" and returns
// the chosen mode. Cancelled selections propagate as ui.ErrCancelled.
func askForMode(harnessName string) (config.Mode, error) {
	opts := []ui.Option{
		{Label: "headless", Hint: "one-shot answer here"},
		{Label: "interactive", Hint: "hand over the terminal"},
		{Label: "cancel", Hint: "do nothing"},
	}
	idx, err := ui.Select(fmt.Sprintf("? open %s…", harnessName), opts, 0)
	if err != nil {
		return "", err
	}
	switch idx {
	case 0:
		return config.ModeHeadless, nil
	case 1:
		return config.ModeInteractive, nil
	default:
		return "", ui.ErrCancelled
	}
}
