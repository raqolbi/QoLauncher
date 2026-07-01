package supervisor

import (
	"errors"
	"os/exec"
	"syscall"
)

// ExitResult describes how a child process terminated.
type ExitResult struct {
	Code   int
	Signal syscall.Signal
	Err    error
}

func (r ExitResult) Failed() bool {
	return r.Code != 0 || r.Signal != 0
}

func (r ExitResult) SignalName() string {
	if r.Signal == 0 {
		return ""
	}
	return r.Signal.String()
}

// ParseExit interprets exec.Cmd Wait error into ExitResult.
func ParseExit(err error) ExitResult {
	if err == nil {
		return ExitResult{Code: 0}
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				sig := status.Signal()
				return ExitResult{Code: 128 + int(sig), Signal: sig}
			}
			return ExitResult{Code: status.ExitStatus()}
		}
	}

	return ExitResult{Code: 1, Err: err}
}
