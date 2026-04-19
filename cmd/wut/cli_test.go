package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withXDGConfigHome points wut at an isolated config dir for the
// duration of a subtest so CLI tests can't read or write the real user's
// config file.
func withXDGConfigHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

// runCLI runs a root-level command with the given args and captures both
// os.Stdout (most commands write via fmt.Printf/Println directly) and
// Cobra's writer (error messages). Each call builds a fresh root so state
// doesn't leak between tests.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	root.AddCommand(NewVersionCmd())
	root.AddCommand(NewInitCmd())
	root.AddCommand(NewHarnessCmd())
	root.AddCommand(NewModeCmd())
	root.AddCommand(NewConfigCmd())
	root.AddCommand(NewSetupCmd())

	// Pipe os.Stdout into a buffer for the duration of this call.
	r, w, _ := os.Pipe()
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	var cobraBuf bytes.Buffer
	root.SetOut(&cobraBuf)
	root.SetErr(&cobraBuf)
	root.SetArgs(args)
	err := root.Execute()

	w.Close()
	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(r)

	return stdoutBuf.String() + cobraBuf.String(), err
}

func TestVersionCommand(t *testing.T) {
	out, err := runCLI(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != Version {
		t.Errorf("version = %q, want %q", out, Version)
	}
}

func TestInitPrintsAllThreeShells(t *testing.T) {
	for _, shell := range []string{"zsh", "bash", "fish"} {
		out, err := runCLI(t, "init", shell)
		if err != nil {
			t.Fatalf("init %s: %v", shell, err)
		}
		if out == "" {
			t.Errorf("%s snippet empty", shell)
			continue
		}
		// Same invariant the shell-package tests check, re-verified at the CLI
		// boundary to catch wiring regressions.
		if !strings.Contains(out, "wut detect --line") {
			t.Errorf("%s snippet missing hook call:\n%s", shell, out)
		}
	}
}

func TestHarnessListShowsPresets(t *testing.T) {
	withXDGConfigHome(t)
	out, err := runCLI(t, "harness", "list")
	if err != nil {
		t.Fatalf("harness list: %v", err)
	}
	for _, want := range []string{"claude", "aider", "codex"} {
		if !strings.Contains(out, want) {
			t.Errorf("harness list missing preset %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "* claude") {
		t.Errorf("harness list should mark claude as active by default:\n%s", out)
	}
}

func TestHarnessAddAndUseRoundTrip(t *testing.T) {
	dir := withXDGConfigHome(t)
	if _, err := runCLI(t, "harness", "add", "custom", "--command", "/bin/echo", "--args", "x,{prompt}"); err != nil {
		t.Fatalf("harness add: %v", err)
	}
	out, _ := runCLI(t, "harness", "list")
	if !strings.Contains(out, "custom") {
		t.Fatalf("added harness not in list:\n%s", out)
	}
	if _, err := runCLI(t, "harness", "use", "custom"); err != nil {
		t.Fatalf("harness use: %v", err)
	}
	out, _ = runCLI(t, "harness", "list")
	if !strings.Contains(out, "* custom") {
		t.Fatalf("active flag didn't move to custom:\n%s", out)
	}
	// Config file on disk should reflect the change.
	raw, _ := os.ReadFile(filepath.Join(dir, "wut", "config.toml"))
	if !strings.Contains(string(raw), `active_harness = "custom"`) {
		t.Errorf("config.toml doesn't show active_harness=custom:\n%s", raw)
	}
}

func TestConfigSetGetRoundTrip(t *testing.T) {
	withXDGConfigHome(t)
	if _, err := runCLI(t, "config", "set", "default_mode", "headless"); err != nil {
		t.Fatalf("config set: %v", err)
	}
	out, err := runCLI(t, "config", "get", "default_mode")
	if err != nil {
		t.Fatalf("config get: %v", err)
	}
	if strings.TrimSpace(out) != "headless" {
		t.Errorf("config get default_mode = %q, want headless", out)
	}
}

func TestConfigSetRejectsUnknownKey(t *testing.T) {
	withXDGConfigHome(t)
	_, err := runCLI(t, "config", "set", "behavior.ghost", "true")
	if err == nil {
		t.Fatal("config set on unknown key should error")
	}
}

func TestModeSetWritesDefaultMode(t *testing.T) {
	withXDGConfigHome(t)
	if _, err := runCLI(t, "mode", "set", "headless"); err != nil {
		t.Fatalf("mode set: %v", err)
	}
	out, _ := runCLI(t, "config", "get", "default_mode")
	if strings.TrimSpace(out) != "headless" {
		t.Errorf("mode set didn't persist:\n%s", out)
	}
}

func TestModeSetRejectsInvalid(t *testing.T) {
	withXDGConfigHome(t)
	if _, err := runCLI(t, "mode", "set", "bogus"); err == nil {
		t.Fatal("mode set bogus should error")
	}
}

func TestSetupNonInteractive(t *testing.T) {
	dir := withXDGConfigHome(t)
	if _, err := runCLI(t, "setup", "--harness", "codex", "--mode", "headless"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "wut", "config.toml"))
	got := string(raw)
	if !strings.Contains(got, `active_harness = "codex"`) {
		t.Errorf("active_harness not set:\n%s", got)
	}
	if !strings.Contains(got, `default_mode = "headless"`) {
		t.Errorf("default_mode not set:\n%s", got)
	}
}
