package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/sonyaihub/wut/internal/config"
)

// Runner executes a prompt against a harness.
type Runner interface {
	Run(ctx context.Context, prompt string) error
}

// Interactive runs a harness by replacing the current process (syscall.Exec).
// stdin/stdout/stderr and the controlling tty belong to the harness after
// exec; detect's process is gone. Any return from Run means exec failed and
// we never handed control over.
type Interactive struct {
	Invocation *config.Invocation
}

func (r Interactive) Run(ctx context.Context, prompt string) error {
	if r.Invocation == nil {
		return fmt.Errorf("interactive runner: nil invocation")
	}
	// Interactive hands the tty to the child via syscall.Exec, which replaces
	// the current process. That's incompatible with piping bytes into stdin —
	// there's no parent left to do the piping. A harness that needs stdin
	// delivery belongs in a headless block, where we fork+exec and can pump.
	if r.Invocation.Stdin != "" {
		return fmt.Errorf("interactive invocation has stdin set — move this to a [harness.<name>.headless] block; interactive mode exec's into the harness and cannot pipe stdin")
	}
	inv := Substitute(r.Invocation, prompt)

	path, err := exec.LookPath(inv.Command)
	if err != nil {
		return fmt.Errorf("harness binary %q not found on PATH — install it, or run `wut setup` to pick a different harness (%w)", inv.Command, err)
	}

	// argv[0] is the command name; the rest are the configured args.
	argv := append([]string{inv.Command}, inv.Args...)

	envv := os.Environ()
	if len(inv.Env) > 0 {
		envv = mergeEnv(envv, inv.Env)
	}
	return syscall.Exec(path, argv, envv)
}

// mergeEnv overlays overrides onto base, preserving the original order for
// everything not overridden.
func mergeEnv(base []string, overrides map[string]string) []string {
	keys := make(map[string]int, len(base))
	for i, kv := range base {
		if eq := strings.IndexByte(kv, '='); eq >= 0 {
			keys[kv[:eq]] = i
		}
	}
	out := append([]string{}, base...)
	for k, v := range overrides {
		line := k + "=" + v
		if idx, ok := keys[k]; ok {
			out[idx] = line
		} else {
			out = append(out, line)
		}
	}
	return out
}
