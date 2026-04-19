package harness

import (
	"strings"

	"github.com/sonyaihub/wut/internal/config"
)

const promptToken = "{prompt}"

// Substitute replaces every occurrence of {prompt} in the invocation's args,
// stdin, and env values with the given prompt. It returns a fresh copy; the
// source invocation is not mutated.
func Substitute(inv *config.Invocation, prompt string) *config.Invocation {
	out := &config.Invocation{
		Command:    inv.Command,
		Args:       make([]string, len(inv.Args)),
		Stdin:      strings.ReplaceAll(inv.Stdin, promptToken, prompt),
		Stream:     inv.Stream,
		Render:     inv.Render,
		TimeoutSec: inv.TimeoutSec,
	}
	for i, a := range inv.Args {
		out.Args[i] = strings.ReplaceAll(a, promptToken, prompt)
	}
	if len(inv.Env) > 0 {
		out.Env = make(map[string]string, len(inv.Env))
		for k, v := range inv.Env {
			out.Env[k] = strings.ReplaceAll(v, promptToken, prompt)
		}
	}
	return out
}
