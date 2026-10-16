// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package utils

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type commandExecutor struct {
	timeout int
	debug   bool
}

func CommandExecutor(timeout int, debug bool) commandExecutor {
	var t int
	var d bool
	if timeout > 0 {
		t = timeout
	} else {
		t = 60
	}
	if debug {
		d = debug
	} else {
		d = false
	}
	return commandExecutor{t, d}
}

func (ce *commandExecutor) SetDebug(debug bool) {
	ce.debug = debug
}

func (ce commandExecutor) Exec(name string, arg ...string) (string, error) {
	if ce.debug {
		fmt.Printf("Executing: %s %v\n", name, arg)
	}
	out, err := exec.Command(name, arg...).CombinedOutput() // #nosec G204 -- command executor by design runs variable commands
	return ce.finish(out, err)
}

// killWaitDelay bounds how long a timed-out command's Wait may linger
// after the process is killed. Without it, a grandchild that inherited
// the stdout pipe (the docker CLI can leave one behind) keeps
// CombinedOutput blocked even though the direct child is gone.
const killWaitDelay = 5 * time.Second

// ExecWithTimeout runs the command like Exec but abandons it once
// timeout elapses: the process is killed and the returned error wraps
// context.DeadlineExceeded so callers can tell a hang apart from an
// ordinary failure. Whatever the command wrote before the deadline is
// returned as output in both cases.
func (ce commandExecutor) ExecWithTimeout(timeout time.Duration, name string, arg ...string) (string, error) {
	if ce.debug {
		fmt.Printf("Executing (timeout %s): %s %v\n", timeout, name, arg)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, arg...) // #nosec G204 -- command executor by design runs variable commands
	cmd.WaitDelay = killWaitDelay
	out, err := cmd.CombinedOutput()
	if err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = fmt.Errorf("%s timed out after %s: %w", name, timeout, context.DeadlineExceeded)
	}
	return ce.finish(out, err)
}

// finish applies the shared post-processing for both Exec variants:
// debug logging of failures and folding the command's own output into
// the error. A bare exec.ExitError stringifies as just "exit status N",
// which is what callers end up printing - including the combined output
// (stderr is in there) makes failures diagnosable.
func (ce commandExecutor) finish(out []byte, err error) (string, error) {
	if ce.debug && err != nil {
		fmt.Printf("Command failed with error: %v\n", err)
		fmt.Printf("Command output: %s\n", string(out))
	}
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			err = fmt.Errorf("%w: %s", err, msg)
		}
	}
	return string(out), err
}
