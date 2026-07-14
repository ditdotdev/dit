// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package utils

import (
	"errors"
	"testing"
)

// withWSLSeams swaps the goos + getWSLKernel seams for the duration of fn
// and restores them on return. Lets tests drive CheckWSL2AndAdvise on Linux.
func withWSLSeams(t *testing.T, gOOS string, kernel string, kernelErr error, fn func()) {
	t.Helper()
	origGoos, origGet := goos, getWSLKernel
	defer func() { goos, getWSLKernel = origGoos, origGet }()
	goos = gOOS
	getWSLKernel = func() (string, error) { return kernel, kernelErr }
	fn()
}

func TestDetectWSL2_WithWSL2Kernel(t *testing.T) {
	info := DetectWSL2("6.6.87.2-microsoft-standard-WSL2")
	if !info.IsWSL2 {
		t.Error("Expected IsWSL2 to be true")
	}
	if info.KernelMajor != 6 {
		t.Errorf("Expected KernelMajor 6, got %d", info.KernelMajor)
	}
	if info.KernelMinor != 6 {
		t.Errorf("Expected KernelMinor 6, got %d", info.KernelMinor)
	}
	if info.KernelPatch != 87 {
		t.Errorf("Expected KernelPatch 87, got %d", info.KernelPatch)
	}
}

func TestDetectWSL2_WithNonWSL2Kernel(t *testing.T) {
	info := DetectWSL2("6.8.0-106-generic")
	if info.IsWSL2 {
		t.Error("Expected IsWSL2 to be false for generic kernel")
	}
}

func TestSupportsModules_ModernKernel(t *testing.T) {
	info := DetectWSL2("6.6.87.2-microsoft-standard-WSL2")
	if !info.SupportsModules() {
		t.Error("Expected SupportsModules true for 6.6.87")
	}
}

func TestSupportsModules_MinimumKernel(t *testing.T) {
	info := DetectWSL2("6.6.36.3-microsoft-standard-WSL2")
	if !info.SupportsModules() {
		t.Error("Expected SupportsModules true for 6.6.36 (minimum)")
	}
}

func TestSupportsModules_OldKernel(t *testing.T) {
	info := DetectWSL2("5.15.167.4-microsoft-standard-WSL2")
	if info.SupportsModules() {
		t.Error("Expected SupportsModules false for 5.15")
	}
}

func TestSupportsModules_JustBelowMinimum(t *testing.T) {
	info := DetectWSL2("6.6.35.1-microsoft-standard-WSL2")
	if info.SupportsModules() {
		t.Error("Expected SupportsModules false for 6.6.35")
	}
}

func TestSupportsModules_FutureKernel(t *testing.T) {
	info := DetectWSL2("7.0.1.1-microsoft-standard-WSL2")
	if !info.SupportsModules() {
		t.Error("Expected SupportsModules true for 7.x")
	}
}

// CheckWSL2AndAdvise invokes `wsl -e uname -r`. On non-Windows hosts it
// short-circuits at the runtime.GOOS check; on Windows hosts without
// WSL it returns when the exec fails. Either way no panic.
func TestCheckWSL2AndAdvise_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("CheckWSL2AndAdvise panicked: %v", r)
		}
	}()
	CheckWSL2AndAdvise()
}

// The following tests use the goos + getWSLKernel seams so the four
// branches of CheckWSL2AndAdvise are reachable on Linux CI, not just on
// a Windows host with WSL installed.

func TestCheckWSL2AndAdvise_NonWindowsShortCircuits(t *testing.T) {
	called := false
	withWSLSeams(t, "linux", "", nil, func() {
		// Override getWSLKernel side-effect detection: if we hit it,
		// flip the bit. The early return on goos != "windows" should
		// prevent that.
		getWSLKernel = func() (string, error) { called = true; return "", nil }
		CheckWSL2AndAdvise()
	})
	if called {
		t.Error("CheckWSL2AndAdvise called the WSL kernel getter on non-Windows host")
	}
}

func TestCheckWSL2AndAdvise_WSLNotAvailable(t *testing.T) {
	// Windows host, but `wsl -e uname -r` fails (WSL not installed or
	// not running). Should return cleanly.
	withWSLSeams(t, "windows", "", errors.New("wsl: command not found"), func() {
		CheckWSL2AndAdvise()
	})
}

func TestCheckWSL2AndAdvise_NotWSL2Kernel(t *testing.T) {
	// Windows host with a kernel string that doesn't match the WSL2
	// marker — e.g. someone ran `uname -r` over SSH against a real
	// Linux box. Should not print a warning.
	withWSLSeams(t, "windows", "6.8.0-106-generic", nil, func() {
		CheckWSL2AndAdvise()
	})
}

func TestCheckWSL2AndAdvise_OldKernelPrintsWarning(t *testing.T) {
	// Windows host with a WSL2 kernel below the 6.6.36 threshold —
	// the warning print branch fires. We only assert no panic; the
	// banner goes to stdout and isn't worth capturing for coverage.
	withWSLSeams(t, "windows", "5.15.167.4-microsoft-standard-WSL2", nil, func() {
		CheckWSL2AndAdvise()
	})
}

func TestCheckWSL2AndAdvise_ModernKernelSilent(t *testing.T) {
	// Windows host with a modern WSL2 kernel — no warning, no error.
	withWSLSeams(t, "windows", "6.6.87.2-microsoft-standard-WSL2", nil, func() {
		CheckWSL2AndAdvise()
	})
}

// TestRealGetWSLKernel hits the production path on whatever the actual
// host is. On Linux it returns the wsl-command-not-found error; on
// Windows-without-WSL it returns a similar error. Either way it
// exercises the real getter for line coverage.
func TestRealGetWSLKernel(t *testing.T) {
	_, _ = realGetWSLKernel()
}
