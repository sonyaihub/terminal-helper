package main

import (
	"errors"
	"os"
	"os/exec"
)

// propagateExit converts a child-process ExitError into an os.Exit with the
// same exit code, so the shell hook sees the real code rather than Cobra's
// generic 1. Non-exit errors are returned unchanged for Cobra to print.
func propagateExit(err error) error {
	if err == nil {
		return nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		os.Exit(ee.ExitCode())
	}
	return err
}
