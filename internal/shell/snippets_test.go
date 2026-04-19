package shell

import (
	"strings"
	"testing"
)

// Every shell snippet has to satisfy the same invariants: install a hook
// function, guard against recursion if wut falls off PATH, and
// honor the 127-means-passthrough exit contract. These tests pin those so a
// regression in the embedded files (or the embed wiring) surfaces before we
// ship broken shell integrations.

func TestZshSnippet(t *testing.T) {
	s := ZshSnippet()
	checks := []string{
		"command_not_found_handler",     // zsh uses `_handler` with trailing r
		"command -v wut",    // recursion guard
		"wut detect --line", // actual hook call
		"rc -eq 127",                    // 127 is the passthrough sentinel
		"return $rc",                    // propagate handler/harness exit code
	}
	for _, want := range checks {
		if !strings.Contains(s, want) {
			t.Errorf("zsh snippet missing %q", want)
		}
	}
}

func TestBashSnippet(t *testing.T) {
	s := BashSnippet()
	checks := []string{
		"command_not_found_handle",      // bash uses `_handle`, no trailing r
		"command -v wut",    // recursion guard
		"wut detect --line", // hook call
		"BASH_VERSINFO",                 // bash-4+ gate
		"rc -eq 127",
		"return $rc",
	}
	for _, want := range checks {
		if !strings.Contains(s, want) {
			t.Errorf("bash snippet missing %q", want)
		}
	}
	// bash 3.2 on macOS predates the hook entirely; make sure we early-return
	// rather than install a handler that would never fire.
	if !strings.Contains(s, "-lt 4") {
		t.Error("bash snippet should early-return on bash < 4")
	}
}

func TestFishSnippet(t *testing.T) {
	s := FishSnippet()
	checks := []string{
		"fish_command_not_found",        // fish's hook name
		"command -v wut",    // recursion guard
		"wut detect --line", // hook call
		"set -l rc $status",             // fish uses $status, not $?
		"test $rc -eq 127",              // fish test syntax
		"return $rc",
	}
	for _, want := range checks {
		if !strings.Contains(s, want) {
			t.Errorf("fish snippet missing %q", want)
		}
	}
}

func TestSnippetsAreNonEmpty(t *testing.T) {
	// An empty snippet would indicate a broken go:embed. Each of these files
	// should have real content.
	for name, s := range map[string]string{
		"zsh":  ZshSnippet(),
		"bash": BashSnippet(),
		"fish": FishSnippet(),
	} {
		if len(strings.TrimSpace(s)) == 0 {
			t.Errorf("%s snippet is empty — go:embed broken?", name)
		}
	}
}
