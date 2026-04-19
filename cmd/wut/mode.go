package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sonyaihub/wut/internal/config"
)

func NewModeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mode",
		Short: "Change default_mode. (Inspect with `wut config get default_mode`.)",
	}
	cmd.AddCommand(newModeSet())
	return cmd
}

func newModeSet() *cobra.Command {
	return &cobra.Command{
		Use:   "set <interactive|headless|ask>",
		Short: "Set default_mode in the config file.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isKnownMode(args[0]) {
				return fmt.Errorf("mode %q is not one of interactive|headless|ask", args[0])
			}
			cfg, path, err := loadConfig()
			if err != nil {
				return err
			}
			cfg.DefaultMode = config.Mode(args[0])
			if err := writeConfig(path, cfg); err != nil {
				return err
			}
			fmt.Printf("✔ default_mode = %q\n", args[0])
			return nil
		},
	}
}

