// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package clients

import "syscall"

// detachedSysProcAttr returns SysProcAttr values that make a spawned child
// process survive the parent's exit on Unix-like systems. Starting a new
// session detaches the child from the parent's controlling terminal so a
// SIGHUP when dit exits doesn't terminate it.
func detachedSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
