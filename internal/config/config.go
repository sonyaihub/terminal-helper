package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Mode identifies how a harness is invoked.
type Mode string

const (
	ModeInteractive Mode = "interactive"
	ModeHeadless    Mode = "headless" // reserved for M2
	ModeAsk         Mode = "ask"      // reserved for M3
)

// RenderMode controls how headless output is written to the terminal.
type RenderMode string

const (
	RenderRaw      RenderMode = "raw"
	RenderBox      RenderMode = "box"
	RenderMarkdown RenderMode = "markdown" // reserved for M3
)

// Invocation describes how to launch a harness in one mode.
type Invocation struct {
	Command string            `toml:"command"`
	Args    []string          `toml:"args"`
	Stdin   string            `toml:"stdin"`
	Env     map[string]string `toml:"env"`

	// Headless-only fields. Ignored in interactive mode.
	Stream     *bool      `toml:"stream"`
	Render     RenderMode `toml:"render"`
	TimeoutSec int        `toml:"timeout_sec"`
}

// StreamEnabled returns whether the invocation's stdout should stream live.
// Defaults to true when unset (matches spec §7 default).
func (i *Invocation) StreamEnabled() bool {
	if i.Stream == nil {
		return true
	}
	return *i.Stream
}

// HeadlessFallback controls what happens when headless is requested but the
// active harness has no headless block defined.
type HeadlessFallback string

const (
	FallbackInteractive HeadlessFallback = "interactive"
	FallbackAsk         HeadlessFallback = "ask"
	FallbackError       HeadlessFallback = "error"
)

// Harness bundles interactive / headless invocations for a single harness.
type Harness struct {
	Interactive *Invocation `toml:"interactive"`
	Headless    *Invocation `toml:"headless"`
}

// Behavior controls optional classifier/runner knobs.
type Behavior struct {
	Confirm          bool             `toml:"confirm"`
	Passthrough      []string         `toml:"passthrough"`
	Spinner          *bool            `toml:"spinner"`
	SpinnerStyle     string           `toml:"spinner_style"`
	HeadlessFallback HeadlessFallback `toml:"headless_fallback"`
}

// SpinnerEnabled returns whether the headless spinner should draw. Defaults
// to true when the field is unset.
func (b *Behavior) SpinnerEnabled() bool {
	if b.Spinner == nil {
		return true
	}
	return *b.Spinner
}

// Config is the top-level file at ~/.config/wut/config.toml.
type Config struct {
	ActiveHarness string             `toml:"active_harness"`
	DefaultMode   Mode               `toml:"default_mode"`
	Behavior      Behavior           `toml:"behavior"`
	Detection     Detection          `toml:"detection"`
	Harness       map[string]Harness `toml:"harness"`
}

// Detection extends the built-in classifier token sets. Both fields are
// additive — the built-in sets in internal/detect are never removed.
type Detection struct {
	ExtraStopwords      []string `toml:"extra_stopwords"`
	ExtraInterrogatives []string `toml:"extra_interrogatives"`
}

// DefaultPath returns the standard config file path, honoring XDG_CONFIG_HOME.
func DefaultPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "wut", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "wut", "config.toml"), nil
}

// Load reads the config file from the given path. If the file does not exist,
// Load returns the preset defaults with no error — first-run users get a
// working config with the claude preset active.
func Load(path string) (*Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Defaults returns a Config populated with the shipped presets and sensible
// defaults. Users' config files override fields after this.
func Defaults() *Config {
	return &Config{
		ActiveHarness: "claude",
		DefaultMode:   ModeInteractive,
		Behavior: Behavior{
			Confirm:          false,
			SpinnerStyle:     "dots",
			HeadlessFallback: FallbackInteractive,
		},
		Harness: map[string]Harness{
			"claude": {
				Interactive: &Invocation{Command: "claude", Args: []string{"{prompt}"}},
				Headless: &Invocation{
					Command: "claude",
					Args:    []string{"-p", "{prompt}"},
					Render:  RenderBox,
				},
			},
			"codex": {
				Interactive: &Invocation{Command: "codex", Args: []string{"{prompt}"}},
				Headless: &Invocation{
					Command: "codex",
					Args:    []string{"exec", "{prompt}"},
					Render:  RenderBox,
				},
			},
			"opencode": {
				Interactive: &Invocation{Command: "opencode", Args: []string{"{prompt}"}},
				Headless: &Invocation{
					Command: "opencode",
					Args:    []string{"run", "{prompt}"},
					Render:  RenderBox,
				},
			},
		},
	}
}

// Validate enforces the minimum invariants needed before we try to launch.
func (c *Config) Validate() error {
	if c.ActiveHarness == "" {
		return errors.New("config: active_harness is empty")
	}
	if _, ok := c.Harness[c.ActiveHarness]; !ok {
		return fmt.Errorf("config: active_harness %q has no matching [harness.%s] block", c.ActiveHarness, c.ActiveHarness)
	}
	if c.DefaultMode == "" {
		c.DefaultMode = ModeInteractive
	}
	switch c.DefaultMode {
	case ModeInteractive, ModeHeadless, ModeAsk:
	default:
		return fmt.Errorf("config: default_mode %q is not one of interactive|headless|ask", c.DefaultMode)
	}
	if c.Behavior.HeadlessFallback == "" {
		c.Behavior.HeadlessFallback = FallbackInteractive
	}
	switch c.Behavior.HeadlessFallback {
	case FallbackInteractive, FallbackAsk, FallbackError:
	default:
		return fmt.Errorf("config: behavior.headless_fallback %q is not one of interactive|ask|error", c.Behavior.HeadlessFallback)
	}
	return nil
}

// ActiveInvocation returns the invocation block for the active harness in the
// requested mode. Returns an error if the harness or mode is unconfigured.
func (c *Config) ActiveInvocation(mode Mode) (*Invocation, error) {
	h, ok := c.Harness[c.ActiveHarness]
	if !ok {
		return nil, fmt.Errorf("config: no harness named %q", c.ActiveHarness)
	}
	switch mode {
	case ModeInteractive:
		if h.Interactive == nil {
			return nil, fmt.Errorf("harness %q has no interactive block", c.ActiveHarness)
		}
		return h.Interactive, nil
	case ModeHeadless:
		if h.Headless == nil {
			return nil, fmt.Errorf("harness %q has no headless block", c.ActiveHarness)
		}
		return h.Headless, nil
	default:
		return nil, fmt.Errorf("mode %q not supported", mode)
	}
}
