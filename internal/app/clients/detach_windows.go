// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package clients

import "syscall"

// Windows process creation flags.
const (
	// DETACHED_PROCESS: the child does not inherit its parent's console.
	detachedProcess = 0x00000008
	// CREATE_NEW_PROCESS_GROUP: the child runs in a new process group so
	// Ctrl-C in the parent's console doesn't propagate to it.
	createNewProcessGroup = 0x00000200
)

// detachedSysProcAttr returns SysProcAttr values that make a spawned child
// process survive the parent's exit. Needed because the Chocolatey-installed
// `kubectl` is a thin shim PE that spawns the real kubectl.exe and then exits;
// without these flags, when dit exits, the real kubectl.exe also exits, which
// kills `kubectl port-forward` and breaks `psql -h localhost` in the demo.
func detachedSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: detachedProcess | createNewProcessGroup,
		HideWindow:    true,
	}
}
