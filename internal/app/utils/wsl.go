// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package utils

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// WSL2Info contains information about the WSL2 environment
type WSL2Info struct {
	IsWSL2        bool
	KernelVersion string // e.g. "6.6.87.2-microsoft-standard-WSL2"
	KernelMajor   int
	KernelMinor   int
	KernelPatch   int
}

// DetectWSL2 parses a kernel version string and determines if it is a WSL2 kernel.
func DetectWSL2(dockerKernelVersion string) WSL2Info {
	info := WSL2Info{}

	// Check if the Docker kernel contains "microsoft-standard-WSL2"
	if !strings.Contains(dockerKernelVersion, "microsoft-standard-WSL2") {
		return info
	}

	info.IsWSL2 = true
	info.KernelVersion = dockerKernelVersion

	// Parse version: "6.6.87.2-microsoft-standard-WSL2" -> 6, 6, 87
	parts := strings.SplitN(dockerKernelVersion, "-", 2)
	if len(parts) > 0 {
		vParts := strings.Split(parts[0], ".")
		if len(vParts) >= 3 {
			info.KernelMajor, _ = strconv.Atoi(vParts[0])
			info.KernelMinor, _ = strconv.Atoi(vParts[1])
			info.KernelPatch, _ = strconv.Atoi(vParts[2])
		}
	}

	return info
}

// SupportsModules returns true if the WSL2 kernel supports CONFIG_MODULES=y
// (kernel >= 6.6.36)
func (w WSL2Info) SupportsModules() bool {
	if !w.IsWSL2 {
		return false
	}
	if w.KernelMajor > 6 {
		return true
	}
	if w.KernelMajor == 6 && w.KernelMinor > 6 {
		return true
	}
	if w.KernelMajor == 6 && w.KernelMinor == 6 && w.KernelPatch >= 36 {
		return true
	}
	return false
}

// goos and getWSLKernel are test seams. Production callers see
// runtime.GOOS and the wsl shell-out exactly as before; tests can
// swap them to drive CheckWSL2AndAdvise's branches on Linux CI
// without an actual WSL2 environment.
var (
	goos         = runtime.GOOS
	getWSLKernel = realGetWSLKernel
)

func realGetWSLKernel() (string, error) {
	out, err := exec.Command("wsl", "-e", "uname", "-r").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// CheckWSL2AndAdvise prints guidance for WSL2 users during install
func CheckWSL2AndAdvise() {
	if goos != "windows" {
		return
	}

	kernelVersion, err := getWSLKernel()
	if err != nil {
		// WSL not available or not running, not a WSL2 environment
		return
	}

	info := DetectWSL2(kernelVersion)

	if !info.IsWSL2 {
		return
	}

	if !info.SupportsModules() {
		fmt.Println()
		fmt.Println("WARNING: Your WSL2 kernel (" + info.KernelVersion + ") does not support")
		fmt.Println("kernel module loading (requires >= 6.6.36). Please update WSL:")
		fmt.Println()
		fmt.Println("  wsl --update")
		fmt.Println()
		fmt.Println("Then restart and retry: dit install")
		fmt.Println()
	}
}
