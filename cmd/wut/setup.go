package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/spf13/cobra"

	"github.com/sonyabytes/wut/internal/config"
	"github.com/sonyabytes/wut/internal/ui"
)

func NewSetupCmd() *cobra.Command {
	var harnessFlag, modeFlag, shellFlag string
	var installHookFlag, installCompletionFlag bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Guided config wizard: pick harness, default mode, and install the shell hook.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadConfig()
			if err != nil {
				return err
			}

			nonInteractive := harnessFlag != "" && modeFlag != ""
			chosenHarness := harnessFlag
			chosenMode := config.Mode(modeFlag)

			if nonInteractive {
				if _, ok := cfg.Harness[harnessFlag]; !ok {
					return fmt.Errorf("no harness named %q (see `wut harness list`)", harnessFlag)
				}
				if !isKnownMode(modeFlag) {
					return fmt.Errorf("mode %q is not one of interactive|headless|ask", modeFlag)
				}
			} else {
				chosenHarness, err = pickHarness(cfg, harnessFlag)
				if err != nil {
					return err
				}
				if chosenMode == "" {
					chosenMode, err = pickMode()
					if err != nil {
						return err
					}
				} else if !isKnownMode(modeFlag) {
					return fmt.Errorf("mode %q is not one of interactive|headless|ask", modeFlag)
				}
			}

			cfg.ActiveHarness = chosenHarness
			cfg.DefaultMode = chosenMode
			if err := writeConfig(path, cfg); err != nil {
				return err
			}
			fmt.Printf("✔ wrote %s (harness=%s, mode=%s)\n", path, chosenHarness, chosenMode)

			sh := shellFlag
			if sh == "" {
				sh = detectShell()
			}

			doHook := installHookFlag
			if !nonInteractive && !cmd.Flags().Changed("install-hook") {
				ok, confirmErr := ui.Confirm("install shell hook?")
				switch {
				case confirmErr == nil:
					doHook = ok
				case errors.Is(confirmErr, ui.ErrCancelled):
					doHook = false
					// default: no TTY or other prompt failure — preserve installHookFlag (true).
				}
			}
			if doHook {
				if err := installHook(sh, true); err != nil {
					fmt.Fprintf(os.Stderr, "⚠ hook install skipped: %v\n", err)
				}
			}

			doCompletion := installCompletionFlag
			if !nonInteractive && !cmd.Flags().Changed("install-completion") {
				ok, confirmErr := ui.Confirm("install shell completion?")
				switch {
				case confirmErr == nil:
					doCompletion = ok
				case errors.Is(confirmErr, ui.ErrCancelled):
					doCompletion = false
					// default: no TTY or other prompt failure — preserve installCompletionFlag (false).
				}
			}
			if doCompletion {
				if err := installCompletion(cmd.Root(), sh, true, false); err != nil {
					fmt.Fprintf(os.Stderr, "⚠ completion install skipped: %v\n", err)
				}
			}

			fmt.Println("  run `wut doctor` to verify")
			return nil
		},
	}
	cmd.Flags().StringVar(&harnessFlag, "harness", "", "harness name (skips the picker when set)")
	cmd.Flags().StringVar(&modeFlag, "mode", "", "default mode: interactive|headless|ask (skips the picker when set)")
	cmd.Flags().StringVar(&shellFlag, "shell", "", "override shell detection for hook/completion install (zsh|bash|fish)")
	cmd.Flags().BoolVar(&installHookFlag, "install-hook", true, "install the shell hook after writing config (use --install-hook=false to skip)")
	cmd.Flags().BoolVar(&installCompletionFlag, "install-completion", false, "install shell completion after writing config")
	return cmd
}

func pickHarness(cfg *config.Config, preset string) (string, error) {
	names := make([]string, 0, len(cfg.Harness))
	for n := range cfg.Harness {
		names = append(names, n)
	}
	sort.Strings(names)

	initial := 0
	opts := make([]ui.Option, 0, len(names))
	for i, n := range names {
		hint := ""
		if h := cfg.Harness[n].Interactive; h != nil {
			if _, err := exec.LookPath(h.Command); err == nil {
				hint = "detected on PATH"
			} else {
				hint = "not installed"
			}
		}
		opts = append(opts, ui.Option{Label: n, Hint: hint})
		if n == cfg.ActiveHarness || n == preset {
			initial = i
		}
	}

	idx, err := ui.Select("? Which harness do you use?", opts, initial)
	if err != nil {
		return "", err
	}
	return names[idx], nil
}

func pickMode() (config.Mode, error) {
	opts := []ui.Option{
		{Label: "headless", Hint: "prints a one-shot answer, stay in shell"},
		{Label: "interactive", Hint: "opens the harness, replaces the terminal"},
		{Label: "ask", Hint: "prompt me each time"},
	}
	idx, err := ui.Select("? Default mode when natural language is detected?", opts, 0)
	if err != nil {
		return "", err
	}
	switch idx {
	case 0:
		return config.ModeHeadless, nil
	case 1:
		return config.ModeInteractive, nil
	case 2:
		return config.ModeAsk, nil
	}
	return config.ModeInteractive, nil
}

func isKnownMode(s string) bool {
	switch config.Mode(s) {
	case config.ModeInteractive, config.ModeHeadless, config.ModeAsk:
		return true
	}
	return false
}
