package harness

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/sonyaihub/wut/internal/config"
	"github.com/sonyaihub/wut/internal/render"
)

// HeadlessOptions tunes the non-invocation knobs that come from the top-level
// Behavior block plus a friendly label for the renderer.
type HeadlessOptions struct {
	Label           string // harness name, used in box header and spinner label
	SpinnerEnabled  bool
	SpinnerStyle    string
}

// Headless spawns the harness as a child, pipes its stdout through a renderer,
// forwards stderr, and supports SIGINT + timeout per spec §9.
type Headless struct {
	Invocation *config.Invocation
	Opts       HeadlessOptions
}

func (h Headless) Run(ctx context.Context, prompt string) error {
	if h.Invocation == nil {
		return errors.New("headless runner: nil invocation")
	}
	inv := Substitute(h.Invocation, prompt)

	path, err := exec.LookPath(inv.Command)
	if err != nil {
		return fmt.Errorf("harness binary %q not found on PATH — install it, or run `wut setup` to pick a different harness (%w)", inv.Command, err)
	}

	cmdCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, path, inv.Args...)
	cmd.Env = os.Environ()
	if len(inv.Env) > 0 {
		cmd.Env = mergeEnv(cmd.Env, inv.Env)
	}
	// Spec §9: stdin from configured value or /dev/null. Never borrow user stdin.
	if inv.Stdin != "" {
		cmd.Stdin = strings.NewReader(inv.Stdin)
	} else {
		cmd.Stdin = nil
	}
	cmd.Stderr = os.Stderr
	// Put the child in its own process group so we can signal the whole tree —
	// shells don't forward SIGTERM to foreground children by default, and the
	// timeout watchdog below relies on reaching grandchildren like `sleep`.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	pgid := cmd.Process.Pid // Setpgid=true makes pgid == pid

	// Timeout watchdog — SIGTERM on expiry, SIGKILL 2s later.
	if inv.TimeoutSec > 0 {
		timer := time.AfterFunc(time.Duration(inv.TimeoutSec)*time.Second, func() {
			_ = syscall.Kill(-pgid, syscall.SIGTERM)
			time.AfterFunc(2*time.Second, func() {
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
			})
		})
		defer timer.Stop()
	}

	// SIGINT handling — spec §9: first Ctrl-C forwards SIGINT to child, second
	// within 500ms escalates to SIGKILL.
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT)
	defer signal.Stop(sigCh)
	go func() {
		var lastInt time.Time
		for sig := range sigCh {
			if sig != syscall.SIGINT {
				continue
			}
			now := time.Now()
			if !lastInt.IsZero() && now.Sub(lastInt) < 500*time.Millisecond {
				_ = syscall.Kill(-pgid, syscall.SIGKILL)
				return
			}
			lastInt = now
			_ = syscall.Kill(-pgid, syscall.SIGINT)
		}
	}()

	// Spinner — only when stdout is a tty and enabled.
	stdoutIsTTY := term.IsTerminal(int(os.Stdout.Fd()))
	var spinner *render.Spinner
	if stdoutIsTTY && h.Opts.SpinnerEnabled && h.Opts.SpinnerStyle != "none" {
		spinner = &render.Spinner{
			W:     os.Stdout,
			Label: fmt.Sprintf("%s is thinking…", h.Opts.Label),
			Style: h.Opts.SpinnerStyle,
		}
		spinner.Start()
	}
	stopSpinner := func() {
		if spinner != nil {
			spinner.Stop()
			spinner = nil
		}
	}

	// Renderer — box by default, raw when streaming or explicitly set.
	renderer := newRenderer(inv, os.Stdout, h.Opts.Label)

	// Copy child stdout → renderer. Line-buffered so we can stop the spinner
	// the moment the first byte arrives.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		reader := bufio.NewReader(stdout)
		firstByteSeen := false
		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				if !firstByteSeen {
					stopSpinner()
					firstByteSeen = true
				}
				if _, werr := renderer.Write(buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	waitErr := cmd.Wait()
	wg.Wait()
	stopSpinner()
	_ = renderer.Close()

	return waitErr
}

// newRenderer picks a Writer given the invocation's render preference. Per
// spec §9, markdown is buffered-only; a streaming invocation with
// render=markdown falls back to raw.
func newRenderer(inv *config.Invocation, w io.Writer, label string) render.Writer {
	switch inv.Render {
	case config.RenderRaw:
		return render.Raw{W: w}
	case config.RenderMarkdown:
		if inv.StreamEnabled() {
			return render.Raw{W: w}
		}
		return &render.Markdown{W: w}
	case config.RenderBox, "":
		return &render.Box{W: w, Label: label, Width: detectWidth(w)}
	default:
		return render.Raw{W: w}
	}
}

// detectWidth asks the terminal for a column count; falls back to 80.
func detectWidth(w io.Writer) int {
	type fd interface{ Fd() uintptr }
	if f, ok := w.(fd); ok {
		if width, _, err := term.GetSize(int(f.Fd())); err == nil && width > 0 {
			return width
		}
	}
	return 80
}
