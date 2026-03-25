package utils

import (
	"testing"
)

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

func TestDetectWSL2_WithLinuxKitKernel(t *testing.T) {
	info := DetectWSL2("6.6.71-linuxkit")
	if info.IsWSL2 {
		t.Error("Expected IsWSL2 to be false for linuxkit kernel")
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
