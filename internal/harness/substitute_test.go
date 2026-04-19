package harness

import (
	"reflect"
	"testing"

	"github.com/sonyaihub/wut/internal/config"
)

func TestSubstituteReplacesAllOccurrences(t *testing.T) {
	inv := &config.Invocation{
		Command: "claude",
		Args:    []string{"--msg", "{prompt}", "note: {prompt}"},
		Stdin:   "the prompt is: {prompt}",
		Env:     map[string]string{"PREFIX": "p={prompt}"},
	}
	out := Substitute(inv, "hello world")
	wantArgs := []string{"--msg", "hello world", "note: hello world"}
	if !reflect.DeepEqual(out.Args, wantArgs) {
		t.Fatalf("args: got %v want %v", out.Args, wantArgs)
	}
	if out.Stdin != "the prompt is: hello world" {
		t.Fatalf("stdin: %q", out.Stdin)
	}
	if out.Env["PREFIX"] != "p=hello world" {
		t.Fatalf("env: %q", out.Env["PREFIX"])
	}
	// original must not be mutated
	if inv.Args[1] != "{prompt}" {
		t.Fatalf("source invocation mutated")
	}
}

func TestSubstituteLeavesLiteralsAlone(t *testing.T) {
	inv := &config.Invocation{Command: "claude", Args: []string{"--help"}}
	out := Substitute(inv, "ignored")
	if !reflect.DeepEqual(out.Args, []string{"--help"}) {
		t.Fatalf("args mutated: %v", out.Args)
	}
}

func TestSubstitutePreservesHeadlessFields(t *testing.T) {
	trueVal := true
	inv := &config.Invocation{
		Command:    "x",
		Args:       []string{"{prompt}"},
		Stream:     &trueVal,
		Render:     config.RenderBox,
		TimeoutSec: 30,
	}
	out := Substitute(inv, "p")
	if out.Render != config.RenderBox {
		t.Fatalf("render not preserved: %q", out.Render)
	}
	if out.TimeoutSec != 30 {
		t.Fatalf("timeout_sec not preserved: %d", out.TimeoutSec)
	}
	if out.Stream == nil || *out.Stream != true {
		t.Fatalf("stream not preserved: %v", out.Stream)
	}
}
