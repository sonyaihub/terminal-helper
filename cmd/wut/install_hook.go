package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sonyaihub/wut/internal/shell"
	"github.com/sonyaihub/wut/internal/ui"
)

// marker that we grep for when deciding whether the rc file is already wired.
// Kept short and human-readable; any of our install paths writes something
// that contains this string.
const hookMarker = "wut init"

func NewInstallHookCmd() *cobra.Command {
	var shellFlag string
	var yes bool
	cmd := &cobra.Command{
		Use:   "install-hook",
		Short: "Add the shell hook to your rc file. Detects shell automatically.",
		Long: `install-hook wires wut into your interactive shell.

For zsh/bash it appends a one-line eval to your rc file. For fish it writes
the hook function to a conf.d file. All operations are idempotent.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sh := shellFlag
			if sh == "" {
				sh = detectShell()
			}
			switch sh {
			case "zsh":
				return installRcLine("zsh", rcPath("zsh"), `eval "$(wut init zsh)"`, yes)
			case "bash":
				return installRcLine("bash", rcPath("bash"), `eval "$(wut init bash)"`, yes)
			case "fish":
				return installFishConfD(yes)
			default:
				return fmt.Errorf("could not detect a supported shell (got %q) — pass --shell zsh|bash|fish", sh)
			}
		},
	}
	cmd.Flags().StringVar(&shellFlag, "shell", "", "override shell detection (zsh|bash|fish)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the Y/n confirmation")
	return cmd
}

func detectShell() string {
	s := os.Getenv("SHELL")
	if s == "" {
		return ""
	}
	return filepath.Base(s)
}

func rcPath(sh string) string {
	home, _ := os.UserHomeDir()
	switch sh {
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "bash":
		// Prefer ~/.bashrc since that's where interactive config usually lives;
		// on macOS users may also have ~/.bash_profile sourcing .bashrc.
		return filepath.Join(home, ".bashrc")
	}
	return ""
}

func installRcLine(sh, path, line string, yes bool) error {
	if path == "" {
		return fmt.Errorf("no rc path known for shell %q", sh)
	}
	existing, _ := os.ReadFile(path)
	if strings.Contains(string(existing), hookMarker) {
		fmt.Printf("✓ %s already contains a wut hook — nothing to do\n", path)
		fmt.Printf("  open a new shell or run: source %s\n", path)
		return nil
	}

	if !yes {
		ok, err := ui.Confirm(fmt.Sprintf("append hook to %s?", path))
		if err == ui.ErrCancelled || (err == nil && !ok) {
			fmt.Println("cancelled — no changes made")
			return nil
		}
		// err != nil and != ErrCancelled → no tty; proceed silently so
		// curl | sh style flows still work.
	}

	// Ensure the file exists; touching it is fine.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var block strings.Builder
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		block.WriteByte('\n')
	}
	block.WriteString("\n# added by `wut install-hook`\n")
	block.WriteString(line)
	block.WriteByte('\n')
	if _, err := f.WriteString(block.String()); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	fmt.Printf("✔ appended hook to %s\n", path)
	fmt.Printf("  open a new shell or run: source %s\n", path)
	return nil
}

func installFishConfD(yes bool) error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "fish", "conf.d")
	path := filepath.Join(dir, "wut.fish")

	if _, err := os.Stat(path); err == nil {
		fmt.Printf("✓ %s already exists — nothing to do\n", path)
		fmt.Printf("  open a new fish shell to pick up changes\n")
		return nil
	}

	if !yes {
		ok, err := ui.Confirm(fmt.Sprintf("write fish hook to %s?", path))
		if err == ui.ErrCancelled || (err == nil && !ok) {
			fmt.Println("cancelled — no changes made")
			return nil
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	header := "# added by `wut install-hook`\n"
	if err := os.WriteFile(path, []byte(header+shell.FishSnippet()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("✔ wrote %s\n", path)
	fmt.Printf("  open a new fish shell to pick up changes\n")
	return nil
}
