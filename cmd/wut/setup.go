package main

import (
	"fmt"
	"os/exec"
	"sort"

	"github.com/spf13/cobra"

	"github.com/sonyaihub/wut/internal/config"
	"github.com/sonyaihub/wut/internal/ui"
)

func NewSetupCmd() *cobra.Command {
	var harnessFlag, modeFlag string
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Guided config wizard: pick harness and default mode.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadConfig()
			if err != nil {
				return err
			}

			// Non-interactive path — both flags supplied, no prompts needed.
			if harnessFlag != "" && modeFlag != "" {
				if _, ok := cfg.Harness[harnessFlag]; !ok {
					return fmt.Errorf("no harness named %q (see `wut harness list`)", harnessFlag)
				}
				if !isKnownMode(modeFlag) {
					return fmt.Errorf("mode %q is not one of interactive|headless|ask", modeFlag)
				}
				cfg.ActiveHarness = harnessFlag
				cfg.DefaultMode = config.Mode(modeFlag)
				if err := writeConfig(path, cfg); err != nil {
					return err
				}
				fmt.Printf("✔ wrote %s (harness=%s, mode=%s)\n", path, harnessFlag, modeFlag)
				return nil
			}

			// Interactive path.
			chosenHarness, err := pickHarness(cfg, harnessFlag)
			if err != nil {
				return err
			}
			chosenMode := config.Mode(modeFlag)
			if chosenMode == "" {
				chosenMode, err = pickMode()
				if err != nil {
					return err
				}
			} else if !isKnownMode(modeFlag) {
				return fmt.Errorf("mode %q is not one of interactive|headless|ask", modeFlag)
			}

			cfg.ActiveHarness = chosenHarness
			cfg.DefaultMode = chosenMode
			if err := writeConfig(path, cfg); err != nil {
				return err
			}
			fmt.Printf("✔ wrote %s (harness=%s, mode=%s)\n", path, chosenHarness, chosenMode)
			fmt.Println("  run `wut doctor` to verify")
			return nil
		},
	}
	cmd.Flags().StringVar(&harnessFlag, "harness", "", "harness name (skips the picker when set)")
	cmd.Flags().StringVar(&modeFlag, "mode", "", "default mode: interactive|headless|ask (skips the picker when set)")
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
