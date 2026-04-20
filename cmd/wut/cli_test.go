package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sonyabytes/wut/internal/config"
	"github.com/sonyabytes/wut/internal/detect"
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

// withIsolatedHome points $HOME (and by extension os.UserHomeDir) at a temp
// dir and pins $SHELL to zsh so hook/completion install tests can't touch
// the real user's rc files. Use this on top of withXDGConfigHome for any
// test that exercises install-hook or install-completion.
func withIsolatedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	return home
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
	root.AddCommand(NewKeywordsCmd())
	root.AddCommand(NewCompletionCmd())
	root.AddCommand(NewWhyCmd())

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
	for _, want := range []string{"claude", "codex", "opencode"} {
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

func TestModeGetReflectsDefaultMode(t *testing.T) {
	withXDGConfigHome(t)
	out, err := runCLI(t, "mode", "get")
	if err != nil {
		t.Fatalf("mode get: %v", err)
	}
	if strings.TrimSpace(out) != "interactive" {
		t.Errorf("mode get default = %q, want interactive", out)
	}
	if _, err := runCLI(t, "mode", "set", "headless"); err != nil {
		t.Fatalf("mode set: %v", err)
	}
	out, err = runCLI(t, "mode", "get")
	if err != nil {
		t.Fatalf("mode get after set: %v", err)
	}
	if strings.TrimSpace(out) != "headless" {
		t.Errorf("mode get after set = %q, want headless", out)
	}
}

func TestModeSetRejectsInvalid(t *testing.T) {
	withXDGConfigHome(t)
	if _, err := runCLI(t, "mode", "set", "bogus"); err == nil {
		t.Fatal("mode set bogus should error")
	}
}

func TestKeywordsAddFirstWordAndListRoundTrip(t *testing.T) {
	dir := withXDGConfigHome(t)
	if _, err := runCLI(t, "keywords", "add", "deploy", "--first-word"); err != nil {
		t.Fatalf("keywords add: %v", err)
	}
	out, err := runCLI(t, "keywords", "list")
	if err != nil {
		t.Fatalf("keywords list: %v", err)
	}
	if !strings.Contains(out, "- deploy") {
		t.Errorf("list missing added keyword:\n%s", out)
	}
	// Persisted in the correct section of the TOML file.
	raw, err := os.ReadFile(filepath.Join(dir, "wut", "config.toml"))
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	got := string(raw)
	if !strings.Contains(got, "extra_interrogatives") || !strings.Contains(got, "deploy") {
		t.Errorf("config.toml missing extra_interrogatives=deploy:\n%s", got)
	}
}

func TestKeywordsAddAnywhereWritesStopwords(t *testing.T) {
	dir := withXDGConfigHome(t)
	if _, err := runCLI(t, "keywords", "add", "pls", "--anywhere"); err != nil {
		t.Fatalf("keywords add: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "wut", "config.toml"))
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	got := string(raw)
	if !strings.Contains(got, "extra_stopwords") || !strings.Contains(got, "pls") {
		t.Errorf("config.toml missing extra_stopwords=pls:\n%s", got)
	}
}

func TestKeywordsAddIsIdempotent(t *testing.T) {
	withXDGConfigHome(t)
	if _, err := runCLI(t, "keywords", "add", "deploy", "--first-word"); err != nil {
		t.Fatalf("first add: %v", err)
	}
	out, err := runCLI(t, "keywords", "add", "deploy", "--first-word")
	if err != nil {
		t.Fatalf("second add: %v", err)
	}
	if !strings.Contains(out, "already in") {
		t.Errorf("second add should report duplicate, got:\n%s", out)
	}
}

func TestKeywordsAddRejectsConflictingFlags(t *testing.T) {
	withXDGConfigHome(t)
	if _, err := runCLI(t, "keywords", "add", "foo", "--first-word", "--anywhere"); err == nil {
		t.Fatal("expected error for both flags, got nil")
	}
}

func TestKeywordsAddRejectsWhitespace(t *testing.T) {
	withXDGConfigHome(t)
	if _, err := runCLI(t, "keywords", "add", "two words", "--first-word"); err == nil {
		t.Fatal("whitespace keyword should error")
	}
}

func TestKeywordsRemoveDisambiguatesWhenInBothSets(t *testing.T) {
	withXDGConfigHome(t)
	if _, err := runCLI(t, "keywords", "add", "shared", "--first-word"); err != nil {
		t.Fatalf("add first-word: %v", err)
	}
	if _, err := runCLI(t, "keywords", "add", "shared", "--anywhere"); err != nil {
		t.Fatalf("add anywhere: %v", err)
	}
	// Without a flag the command must refuse.
	if _, err := runCLI(t, "keywords", "remove", "shared"); err == nil {
		t.Fatal("remove without flag in both-sets case should error")
	}
	// Targeted removal leaves the other set intact.
	if _, err := runCLI(t, "keywords", "remove", "shared", "--first-word"); err != nil {
		t.Fatalf("remove --first-word: %v", err)
	}
	out, _ := runCLI(t, "keywords", "list")
	firstIdx := strings.Index(out, "First-word triggers")
	anywhereIdx := strings.Index(out, "Anywhere signals")
	if firstIdx < 0 || anywhereIdx < 0 || firstIdx > anywhereIdx {
		t.Fatalf("list output not in expected shape:\n%s", out)
	}
	firstSection := out[firstIdx:anywhereIdx]
	anywhereSection := out[anywhereIdx:]
	if strings.Contains(firstSection, "shared") {
		t.Errorf("shared should be gone from first-word section:\n%s", firstSection)
	}
	if !strings.Contains(anywhereSection, "shared") {
		t.Errorf("shared should remain in anywhere section:\n%s", anywhereSection)
	}
}

func TestKeywordsRemoveMissingErrors(t *testing.T) {
	withXDGConfigHome(t)
	if _, err := runCLI(t, "keywords", "remove", "ghost"); err == nil {
		t.Fatal("removing nonexistent keyword should error")
	}
}

// TestKeywordsDriveClassifier proves the config written by `keywords add`
// actually flips the classifier's decision — catches wiring regressions
// between the CLI, config serialisation, and internal/detect.
func TestKeywordsDriveClassifier(t *testing.T) {
	withXDGConfigHome(t)
	if _, err := runCLI(t, "keywords", "add", "deploy", "--first-word"); err != nil {
		t.Fatalf("add first-word: %v", err)
	}
	if _, err := runCLI(t, "keywords", "add", "pls", "--anywhere"); err != nil {
		t.Fatalf("add anywhere: %v", err)
	}
	path, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	opts := detect.Options{
		ExtraInterrogatives: cfg.Detection.ExtraInterrogatives,
		ExtraStopwords:      cfg.Detection.ExtraStopwords,
	}
	// "deploy the service pls" has 0 signals before; with both extras it
	// now carries an interrogative (deploy) and a stopword (pls) → Route.
	if got := detect.Classify("deploy the service pls", opts); got != detect.Route {
		t.Errorf("classifier didn't pick up added keywords; got %v", got)
	}
}

func TestHarnessRemoveNonActive(t *testing.T) {
	dir := withXDGConfigHome(t)
	if _, err := runCLI(t, "harness", "add", "custom", "--command", "/bin/echo"); err != nil {
		t.Fatalf("harness add: %v", err)
	}
	if _, err := runCLI(t, "harness", "remove", "custom"); err != nil {
		t.Fatalf("harness remove: %v", err)
	}
	out, _ := runCLI(t, "harness", "list")
	if strings.Contains(out, "custom") {
		t.Errorf("removed harness still appears in list:\n%s", out)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "wut", "config.toml"))
	if strings.Contains(string(raw), "custom") {
		t.Errorf("removed harness still in config.toml:\n%s", raw)
	}
}

func TestHarnessRemoveActiveRequiresForce(t *testing.T) {
	withXDGConfigHome(t)
	_, err := runCLI(t, "harness", "remove", "claude")
	if err == nil {
		t.Fatal("expected error removing active harness without --force")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should mention --force, got: %v", err)
	}
}

func TestHarnessRemoveActiveWithForceSwitchesToNext(t *testing.T) {
	dir := withXDGConfigHome(t)
	// Default active is claude; alphabetically next among presets is codex.
	out, err := runCLI(t, "harness", "remove", "claude", "--force")
	if err != nil {
		t.Fatalf("harness remove --force: %v", err)
	}
	if !strings.Contains(out, "codex") {
		t.Errorf("expected active_harness to switch to codex, output:\n%s", out)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "wut", "config.toml"))
	if !strings.Contains(string(raw), `active_harness = "codex"`) {
		t.Errorf("config.toml should have active_harness=codex:\n%s", raw)
	}
	if strings.Contains(string(raw), `[harness.claude]`) {
		t.Errorf("claude harness should be gone from config.toml:\n%s", raw)
	}
}

func TestHarnessRemoveLastHarnessRefused(t *testing.T) {
	// config.Load merges the on-disk harness map with Defaults(), so the
	// "only remaining harness" path cannot be reached purely through the CLI
	// (preset harnesses always re-appear after a load). Test the helper
	// directly — it is the exact code guarding that error path.
	only := map[string]config.Harness{
		"solo": {Interactive: &config.Invocation{Command: "/bin/echo", Args: []string{"{prompt}"}}},
	}
	_, err := nextHarnessAfterRemoval(only, "solo")
	if err == nil {
		t.Fatal("expected error when removing the only remaining harness")
	}
	if !strings.Contains(err.Error(), "only remaining harness") {
		t.Errorf("error should mention only remaining harness, got: %v", err)
	}
}

func TestHarnessRemoveUnknownErrors(t *testing.T) {
	withXDGConfigHome(t)
	_, err := runCLI(t, "harness", "remove", "ghost")
	if err == nil {
		t.Fatal("expected error removing nonexistent harness")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("error should mention harness name, got: %v", err)
	}
}

func TestSetupNonInteractive(t *testing.T) {
	dir := withXDGConfigHome(t)
	withIsolatedHome(t)
	if _, err := runCLI(t, "setup", "--harness", "codex", "--mode", "headless", "--install-hook=false"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "wut", "config.toml"))
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	got := string(raw)
	if !strings.Contains(got, `active_harness = "codex"`) {
		t.Errorf("active_harness not set:\n%s", got)
	}
	if !strings.Contains(got, `default_mode = "headless"`) {
		t.Errorf("default_mode not set:\n%s", got)
	}
}

func TestSetupNonInteractiveInstallsHookByDefault(t *testing.T) {
	withXDGConfigHome(t)
	home := withIsolatedHome(t)
	if _, err := runCLI(t, "setup", "--harness", "codex", "--mode", "headless"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatalf("read .zshrc: %v", err)
	}
	if !strings.Contains(string(raw), `eval "$(wut init zsh)"`) {
		t.Errorf(".zshrc missing hook line:\n%s", raw)
	}
}

func TestSetupNonInteractiveRespectsInstallHookFalse(t *testing.T) {
	withXDGConfigHome(t)
	home := withIsolatedHome(t)
	if _, err := runCLI(t, "setup", "--harness", "codex", "--mode", "headless", "--install-hook=false"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".zshrc")); !os.IsNotExist(err) {
		t.Errorf(".zshrc should not exist when --install-hook=false, got err=%v", err)
	}
}

func TestSetupNonInteractiveInstallsCompletion(t *testing.T) {
	withXDGConfigHome(t)
	home := withIsolatedHome(t)
	if _, err := runCLI(t, "setup", "--harness", "codex", "--mode", "headless", "--install-hook=false", "--install-completion"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	script, err := os.ReadFile(filepath.Join(home, ".zsh", "completions", "_wut"))
	if err != nil {
		t.Fatalf("read completion: %v", err)
	}
	if !strings.Contains(string(script), "_wut") {
		t.Errorf("completion script missing _wut:\n%s", script)
	}
	rc, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatalf("read .zshrc: %v", err)
	}
	if !strings.Contains(string(rc), "fpath=(~/.zsh/completions $fpath)") {
		t.Errorf(".zshrc missing fpath line:\n%s", rc)
	}
}

func TestSetupCompletionIsIdempotent(t *testing.T) {
	withXDGConfigHome(t)
	home := withIsolatedHome(t)
	args := []string{"setup", "--harness", "codex", "--mode", "headless", "--install-hook=false", "--install-completion"}
	if _, err := runCLI(t, args...); err != nil {
		t.Fatalf("first setup: %v", err)
	}
	if _, err := runCLI(t, args...); err != nil {
		t.Fatalf("second setup: %v", err)
	}
	rc, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	if n := strings.Count(string(rc), "fpath=(~/.zsh/completions $fpath)"); n != 1 {
		t.Errorf("fpath line appears %d times, want 1:\n%s", n, rc)
	}
}

// TestInstallHookCommandRemoved guards against a future accidental
// re-registration of the standalone `wut install-hook` command that m7
// intentionally removed. Setup is the only supported onboarding surface.
func TestInstallHookCommandRemoved(t *testing.T) {
	withXDGConfigHome(t)
	_, err := runCLI(t, "install-hook")
	if err == nil {
		t.Fatal("`wut install-hook` should no longer be a command")
	}
}

func TestWhyRouteAfterKeywordAdd(t *testing.T) {
	withXDGConfigHome(t)
	if _, err := runCLI(t, "keywords", "add", "deploy", "--first-word"); err != nil {
		t.Fatalf("keywords add: %v", err)
	}
	out, err := runCLI(t, "why", "deploy the service pls")
	if err != nil {
		t.Fatalf("why: %v", err)
	}
	if !strings.Contains(out, "ROUTE") {
		t.Errorf("expected ROUTE in output:\n%s", out)
	}
	if !strings.Contains(out, "deploy") {
		t.Errorf("expected keyword 'deploy' in output:\n%s", out)
	}
}

func TestWhyPassthroughHardGate(t *testing.T) {
	withXDGConfigHome(t)
	out, err := runCLI(t, "why", "./script.sh foo bar")
	if err != nil {
		t.Fatalf("why: %v", err)
	}
	if !strings.Contains(out, "PASSTHROUGH") {
		t.Errorf("expected PASSTHROUGH in output:\n%s", out)
	}
	if !strings.Contains(out, "hard gate") {
		t.Errorf("expected 'hard gate' mention in output:\n%s", out)
	}
}

func TestCompletionZshContainsFunction(t *testing.T) {
	out, err := runCLI(t, "completion", "zsh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "_wut") {
		t.Errorf("zsh completion missing _wut function:\n%s", out)
	}
}

func TestCompletionBashContainsFunction(t *testing.T) {
	out, err := runCLI(t, "completion", "bash")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "_wut_") {
		t.Errorf("bash completion missing _wut_ prefix:\n%s", out)
	}
}

func TestCompletionFishContainsDirective(t *testing.T) {
	out, err := runCLI(t, "completion", "fish")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "complete -c wut") {
		t.Errorf("fish completion missing 'complete -c wut':\n%s", out)
	}
}

func TestCompletionUnknownShellErrors(t *testing.T) {
	_, err := runCLI(t, "completion", "tcsh")
	if err == nil {
		t.Fatal("completion tcsh should error")
	}
}
