// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package utils

import (
	"fmt"
	"os/exec"
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
	if ce.debug && err != nil {
		fmt.Printf("Command failed with error: %v\n", err)
		fmt.Printf("Command output: %s\n", string(out))
	}
	return string(out), err
}
