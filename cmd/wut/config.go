package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	"github.com/sonyaihub/wut/internal/config"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and edit the config file.",
	}
	cmd.AddCommand(newConfigPath(), newConfigEdit(), newConfigGet(), newConfigSet())
	return cmd
}

func newConfigPath() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path.",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.DefaultPath()
			if err != nil {
				return err
			}
			fmt.Println(p)
			return nil
		},
	}
}

func newConfigEdit() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the config file in $EDITOR.",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.DefaultPath()
			if err != nil {
				return err
			}
			// Ensure the parent dir + file exist so $EDITOR doesn't choke.
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			if _, err := os.Stat(p); os.IsNotExist(err) {
				if err := writeConfig(p, config.Defaults()); err != nil {
					return err
				}
			}
			editor := firstNonEmpty(os.Getenv("VISUAL"), os.Getenv("EDITOR"), "vi")
			c := exec.Command(editor, p)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
}

func newConfigGet() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Print a config value. With no key, prints the resolved config as TOML.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				return toml.NewEncoder(os.Stdout).Encode(cfg)
			}
			val, err := getConfigKey(cfg, args[0])
			if err != nil {
				return err
			}
			fmt.Println(val)
			return nil
		},
	}
}

func newConfigSet() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a top-level config value. Supported keys: active_harness, default_mode, behavior.confirm, behavior.spinner, behavior.headless_fallback, behavior.spinner_style.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadConfig()
			if err != nil {
				return err
			}
			if err := setConfigKey(cfg, args[0], args[1]); err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				return err
			}
			return writeConfig(path, cfg)
		},
	}
}

// getConfigKey / setConfigKey handle the small set of scalar knobs users
// commonly reach for from the CLI. Anything more exotic belongs in
// `config edit` rather than growing a full expression parser here.
func getConfigKey(c *config.Config, key string) (string, error) {
	switch key {
	case "active_harness":
		return c.ActiveHarness, nil
	case "default_mode":
		return string(c.DefaultMode), nil
	case "behavior.confirm":
		return strconv.FormatBool(c.Behavior.Confirm), nil
	case "behavior.spinner":
		return strconv.FormatBool(c.Behavior.SpinnerEnabled()), nil
	case "behavior.spinner_style":
		return c.Behavior.SpinnerStyle, nil
	case "behavior.headless_fallback":
		return string(c.Behavior.HeadlessFallback), nil
	}
	return "", fmt.Errorf("unknown key %q — try `wut config get` (no arg) to see the full config", key)
}

func setConfigKey(c *config.Config, key, value string) error {
	switch key {
	case "active_harness":
		if _, ok := c.Harness[value]; !ok {
			return fmt.Errorf("no harness named %q — run `wut harness list`", value)
		}
		c.ActiveHarness = value
	case "default_mode":
		if !isKnownMode(value) {
			return fmt.Errorf("mode %q is not one of interactive|headless|ask", value)
		}
		c.DefaultMode = config.Mode(value)
	case "behavior.confirm":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("behavior.confirm expects true|false, got %q", value)
		}
		c.Behavior.Confirm = b
	case "behavior.spinner":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("behavior.spinner expects true|false, got %q", value)
		}
		c.Behavior.Spinner = &b
	case "behavior.spinner_style":
		c.Behavior.SpinnerStyle = value
	case "behavior.headless_fallback":
		c.Behavior.HeadlessFallback = config.HeadlessFallback(value)
	default:
		return fmt.Errorf("unknown or read-only key %q", key)
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
