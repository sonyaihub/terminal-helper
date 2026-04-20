package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sonyabytes/wut/internal/ui"
)

// Guards the fpath block installCompletion adds to ~/.zshrc so reruns are no-ops.
// Distinct from hookMarker so hook and completion can be toggled independently.
const completionMarker = "wut setup` — completion"

// installCompletion writes a shell completion script for sh to the appropriate
// path. For zsh it also idempotently adds an fpath= line to ~/.zshrc so the
// script is actually picked up. With printOnly, dumps to stdout instead.
func installCompletion(root *cobra.Command, sh string, yes, printOnly bool) error {
	switch sh {
	case "zsh":
		return installZshCompletion(root, yes, printOnly)
	case "bash":
		return installBashCompletion(root, yes, printOnly)
	case "fish":
		return installFishCompletion(root, yes, printOnly)
	case "":
		return fmt.Errorf("could not detect shell — set $SHELL or pass --shell zsh|bash|fish")
	default:
		return fmt.Errorf("unsupported shell %q — supported: zsh, bash, fish", sh)
	}
}

func installZshCompletion(root *cobra.Command, yes, printOnly bool) error {
	if printOnly {
		return root.GenZshCompletion(os.Stdout)
	}
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".zsh", "completions")
	path := filepath.Join(dir, "_wut")

	if !yes {
		ok, err := ui.Confirm(fmt.Sprintf("write zsh completion to %s?", path))
		if errors.Is(err, ui.ErrCancelled) || (err == nil && !ok) {
			fmt.Println("cancelled — no changes made")
			return nil
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := root.GenZshCompletion(f); err != nil {
		f.Close()
		return fmt.Errorf("generate zsh completion: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Printf("✔ wrote %s\n", path)

	rc := rcPath("zsh")
	existing, _ := os.ReadFile(rc)
	if strings.Contains(string(existing), completionMarker) {
		fmt.Printf("✓ %s already sources ~/.zsh/completions\n", rc)
		return nil
	}
	block := fmt.Sprintf("\n# added by `%s\nfpath=(~/.zsh/completions $fpath)\nautoload -U compinit && compinit\n", completionMarker)
	file, err := os.OpenFile(rc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", rc, err)
	}
	defer file.Close()
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		block = "\n" + block
	}
	if _, err := file.WriteString(block); err != nil {
		return fmt.Errorf("write %s: %w", rc, err)
	}
	fmt.Printf("✔ appended fpath to %s\n", rc)
	fmt.Printf("  open a new shell or run: source %s\n", rc)
	return nil
}

func installBashCompletion(root *cobra.Command, yes, printOnly bool) error {
	if printOnly {
		return root.GenBashCompletion(os.Stdout)
	}
	path := bashCompletionPath()

	if !yes {
		ok, err := ui.Confirm(fmt.Sprintf("write bash completion to %s?", path))
		if errors.Is(err, ui.ErrCancelled) || (err == nil && !ok) {
			fmt.Println("cancelled — no changes made")
			return nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := root.GenBashCompletion(f); err != nil {
		f.Close()
		return fmt.Errorf("generate bash completion: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Printf("✔ wrote %s\n", path)
	fmt.Printf("  open a new shell to pick up changes\n")
	return nil
}

// bashCompletionPath prefers Homebrew's bash-completion dir if brew is on
// PATH, else falls back to the XDG user-level spec path.
func bashCompletionPath() string {
	if out, err := exec.Command("brew", "--prefix").Output(); err == nil {
		prefix := strings.TrimSpace(string(out))
		if prefix != "" {
			return filepath.Join(prefix, "etc", "bash_completion.d", "wut")
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "bash-completion", "completions", "wut")
}

func installFishCompletion(root *cobra.Command, yes, printOnly bool) error {
	if printOnly {
		return root.GenFishCompletion(os.Stdout, true)
	}
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "fish", "completions")
	path := filepath.Join(dir, "wut.fish")

	if !yes {
		ok, err := ui.Confirm(fmt.Sprintf("write fish completion to %s?", path))
		if errors.Is(err, ui.ErrCancelled) || (err == nil && !ok) {
			fmt.Println("cancelled — no changes made")
			return nil
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := root.GenFishCompletion(f, true); err != nil {
		f.Close()
		return fmt.Errorf("generate fish completion: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Printf("✔ wrote %s\n", path)
	fmt.Printf("  open a new fish shell to pick up changes\n")
	return nil
}
