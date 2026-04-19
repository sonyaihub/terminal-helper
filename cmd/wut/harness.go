package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	"github.com/sonyaihub/wut/internal/config"
)

func NewHarnessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "harness",
		Short: "Manage harnesses: list, switch, and test.",
	}
	cmd.AddCommand(newHarnessList(), newHarnessUse(), newHarnessTest(), newHarnessAdd())
	return cmd
}

func newHarnessAdd() *cobra.Command {
	var (
		interactiveCmd     string
		interactiveArgs    []string
		headlessCmd        string
		headlessArgs       []string
		headlessRender     string
		headlessStream     bool
		headlessStreamSet  bool
		setActive          bool
	)
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Register a new harness from the CLI.",
		Long: `Add a harness to ~/.config/wut/config.toml.

At minimum, supply --command for interactive mode. Args default to ["{prompt}"].
For a headless block, pass --headless-command (and optional --headless-args,
--headless-render, --headless-stream).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if interactiveCmd == "" {
				return fmt.Errorf("--command is required (interactive mode)")
			}
			cfg, path, err := loadConfig()
			if err != nil {
				return err
			}
			if _, exists := cfg.Harness[name]; exists {
				return fmt.Errorf("harness %q already exists; edit %s or pick another name", name, path)
			}
			if len(interactiveArgs) == 0 {
				interactiveArgs = []string{"{prompt}"}
			}
			h := config.Harness{
				Interactive: &config.Invocation{Command: interactiveCmd, Args: interactiveArgs},
			}
			if headlessCmd != "" {
				if len(headlessArgs) == 0 {
					headlessArgs = []string{"{prompt}"}
				}
				inv := &config.Invocation{
					Command: headlessCmd,
					Args:    headlessArgs,
					Render:  config.RenderMode(headlessRender),
				}
				if headlessStreamSet {
					v := headlessStream
					inv.Stream = &v
				}
				h.Headless = inv
			}
			cfg.Harness[name] = h
			if setActive {
				cfg.ActiveHarness = name
			}
			if err := writeConfig(path, cfg); err != nil {
				return err
			}
			fmt.Printf("✔ added harness %q to %s\n", name, path)
			if setActive {
				fmt.Printf("✔ active_harness set to %q\n", name)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&interactiveCmd, "command", "", "interactive command (required)")
	cmd.Flags().StringSliceVar(&interactiveArgs, "args", nil, "interactive args (default: [\"{prompt}\"])")
	cmd.Flags().StringVar(&headlessCmd, "headless-command", "", "headless command (optional)")
	cmd.Flags().StringSliceVar(&headlessArgs, "headless-args", nil, "headless args (default: [\"{prompt}\"])")
	cmd.Flags().StringVar(&headlessRender, "headless-render", "box", "headless render mode: raw|box|markdown")
	cmd.Flags().BoolVar(&headlessStream, "headless-stream", true, "stream headless output (default true)")
	cmd.Flags().BoolVar(&setActive, "use", false, "also set this harness as active_harness")
	cmd.PreRun = func(c *cobra.Command, _ []string) {
		headlessStreamSet = c.Flags().Changed("headless-stream")
	}
	return cmd
}

func newHarnessList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show all configured harnesses and which one is active.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig()
			if err != nil {
				return err
			}
			names := make([]string, 0, len(cfg.Harness))
			for n := range cfg.Harness {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				marker := "  "
				if n == cfg.ActiveHarness {
					marker = "* "
				}
				h := cfg.Harness[n]
				modes := []string{}
				if h.Interactive != nil {
					modes = append(modes, "interactive")
				}
				if h.Headless != nil {
					modes = append(modes, "headless")
				}
				fmt.Printf("%s%-12s  modes: %v\n", marker, n, modes)
			}
			return nil
		},
	}
}

func newHarnessUse() *cobra.Command {
	var cmdOverride string
	cmd := &cobra.Command{
		Use:   "use <name>",
		Short: "Set the active harness. Optional --command swaps the binary in every mode block (for wrappers like `claude-yolo`).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, path, err := loadConfig()
			if err != nil {
				return err
			}
			h, ok := cfg.Harness[name]
			if !ok {
				return fmt.Errorf("no harness named %q; run `wut harness list` to see options", name)
			}
			if cmdOverride != "" {
				// Accept shell-style "bin --flag --flag2" strings: the first
				// whitespace-delimited token is the binary, the rest are
				// prepended to the preset's existing args so preset arg
				// structure (e.g. `-p {prompt}` for headless) still applies.
				// Quoting isn't handled — use --command plus `config edit`
				// if you need args with embedded spaces.
				parts := strings.Fields(cmdOverride)
				if len(parts) == 0 {
					return fmt.Errorf("--command is empty")
				}
				bin, extra := parts[0], parts[1:]
				if h.Interactive != nil {
					h.Interactive.Command = bin
					h.Interactive.Args = append(append([]string{}, extra...), h.Interactive.Args...)
				}
				if h.Headless != nil {
					h.Headless.Command = bin
					h.Headless.Args = append(append([]string{}, extra...), h.Headless.Args...)
				}
				cfg.Harness[name] = h
			}
			cfg.ActiveHarness = name
			if err := writeConfig(path, cfg); err != nil {
				return err
			}
			fmt.Printf("✔ active_harness = %q\n", name)
			if cmdOverride != "" {
				if h.Interactive != nil {
					fmt.Printf("  interactive: %s %s\n", h.Interactive.Command, strings.Join(h.Interactive.Args, " "))
				}
				if h.Headless != nil {
					fmt.Printf("  headless:    %s %s\n", h.Headless.Command, strings.Join(h.Headless.Args, " "))
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&cmdOverride, "command", "", "override the binary for every mode block of this harness")
	return cmd
}

func newHarnessTest() *cobra.Command {
	var promptFlag string
	var modeFlag string
	cmd := &cobra.Command{
		Use:   "test [name]",
		Short: "Invoke a harness directly (skipping detection) with the given prompt.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig()
			if err != nil {
				return err
			}
			name := cfg.ActiveHarness
			if len(args) == 1 {
				name = args[0]
			}
			mode := cfg.DefaultMode
			if modeFlag != "" {
				mode = config.Mode(modeFlag)
			}
			return propagateExit(runHarness(cmd, cfg, name, mode, promptFlag))
		},
	}
	cmd.Flags().StringVar(&promptFlag, "prompt", "hello from wut", "prompt to pass to the harness")
	cmd.Flags().StringVar(&modeFlag, "mode", "", "override default_mode (interactive|headless)")
	return cmd
}

// loadConfig resolves the config path, loads the file (or defaults if missing),
// and returns both the config and the resolved path so callers can write back.
func loadConfig() (*config.Config, string, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return nil, "", err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, "", err
	}
	return cfg, path, nil
}

func writeConfig(path string, cfg *config.Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".config-*.toml")
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
