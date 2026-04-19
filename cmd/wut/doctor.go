package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sonyaihub/wut/internal/config"
)

func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check config validity, active harness binary, and shell hook install.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ok := true

			path, err := config.DefaultPath()
			if err != nil {
				return err
			}

			if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
				fmt.Printf("- config: not found at %s (using defaults)\n", path)
			} else if statErr != nil {
				fmt.Printf("✗ config: cannot stat %s: %v\n", path, statErr)
				ok = false
			} else {
				fmt.Printf("✓ config: %s\n", path)
			}

			cfg, err := config.Load(path)
			if err != nil {
				fmt.Printf("✗ config: %v\n", err)
				return err
			}

			fmt.Printf("✓ active_harness = %q, default_mode = %q\n", cfg.ActiveHarness, cfg.DefaultMode)

			inv, err := cfg.ActiveInvocation(config.ModeInteractive)
			if err != nil {
				fmt.Printf("✗ interactive invocation: %v\n", err)
				ok = false
			} else {
				if p, err := exec.LookPath(inv.Command); err != nil {
					fmt.Printf("✗ harness binary %q not found on PATH\n", inv.Command)
					ok = false
				} else {
					fmt.Printf("✓ harness binary: %s\n", p)
				}
			}

			if hookOK, hint := checkHookInstall(); hookOK {
				fmt.Printf("✓ shell hook: %s\n", hint)
			} else {
				fmt.Printf("- shell hook: %s\n", hint)
				fmt.Printf("  run `wut install-hook` to wire it up\n")
			}

			if !ok {
				return errors.New("doctor found issues")
			}
			return nil
		},
	}
}

// checkHookInstall looks for evidence of our hook in the rc file that matches
// the user's $SHELL. Returns (installed, human-readable hint). We intentionally
// do not fail on "not found" — a user can legitimately source the hook from
// some other file (company dotfiles, chezmoi, nix-home-manager) we can't see.
func checkHookInstall() (bool, string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, "could not resolve $HOME"
	}
	sh := filepath.Base(os.Getenv("SHELL"))
	var candidate string
	switch sh {
	case "zsh":
		candidate = filepath.Join(home, ".zshrc")
	case "bash":
		candidate = filepath.Join(home, ".bashrc")
	case "fish":
		candidate = filepath.Join(home, ".config", "fish", "conf.d", "wut.fish")
	default:
		return false, fmt.Sprintf("unrecognized $SHELL (%q) — not checking rc file", sh)
	}
	data, err := os.ReadFile(candidate)
	if err != nil {
		return false, fmt.Sprintf("%s not readable or missing", candidate)
	}
	// install-hook leaves this marker; handwritten `eval "$(wut init
	// zsh)"` also matches because `wut init` is part of both.
	if strings.Contains(string(data), "wut init") {
		return true, candidate
	}
	return false, fmt.Sprintf("%s exists but contains no wut hook", candidate)
}
