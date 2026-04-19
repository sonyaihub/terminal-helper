package main

import (
	"github.com/spf13/cobra"

	"github.com/sonyaihub/wut/internal/config"
)

// NewRunCmd skips detection entirely and always launches the active harness
// with the given prompt. Intended for shell key-bindings and scripts.
func NewRunCmd() *cobra.Command {
	var line, modeFlag string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Launch the active harness with the given prompt (skips detection).",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig()
			if err != nil {
				return err
			}
			mode := cfg.DefaultMode
			if modeFlag != "" {
				mode = config.Mode(modeFlag)
			}
			return propagateExit(runHarness(cmd, cfg, cfg.ActiveHarness, mode, line))
		},
	}
	cmd.Flags().StringVar(&line, "line", "", "prompt to forward to the harness")
	cmd.Flags().StringVar(&modeFlag, "mode", "", "override default_mode (interactive|headless|ask)")
	_ = cmd.MarkFlagRequired("line")
	return cmd
}
