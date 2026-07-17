// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestK8sInstall_DockerNotAvailable(t *testing.T) {
	d := &fakeDocker{versionErr: errors.New("docker missing")}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Install("v1.0.0", "dit", false, 9999, "ctx", nil)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Error checking docker version") {
		t.Errorf("expected docker version error, got %q", output)
	}
}

func TestK8sInstall_TriggersPullWhenNotDownloaded(t *testing.T) {
	d := &fakeDocker{
		latestDownloaded: false,
	}
	_ = captureStdout(func() {
		_, _ = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Install("v1.0.0", "dit", false, 9999, "ctx", nil)
			})
		})
	})

	if d.PullCalls < 1 {
		t.Errorf("expected Pull call, got %d", d.PullCalls)
	}
}

func TestK8sInstall_PullErrorExits(t *testing.T) {
	d := &fakeDocker{
		latestDownloaded: false,
		pullErr:          errors.New("pull failed"),
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Install("v1.0.0", "dit", false, 9999, "ctx", nil)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Error pulling image") {
		t.Errorf("expected pull error, got %q", output)
	}
}

func TestK8sInstall_HappyPathRemovesStaleAndLaunches(t *testing.T) {
	d := &fakeDocker{
		latestDownloaded: true,
		serverAvailable:  true,
		launchAvailable:  true,
	}

	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Install("v1.0.0", "dit", true, 9999, "ctx", nil)
			})
		})
	})

	if d.RemoveCalls < 2 {
		t.Errorf("expected docker.Remove calls for stale server + launch, got %d", d.RemoveCalls)
	}
	if strings.Contains(output, "timed out waiting") {
		t.Errorf("launch-log follow timed out on a FINISHED fixture: %q", output)
	}
	// Regression for #214: cleanup must target the context-derived container
	// names ("dit-<context>-*"), not the hardcoded "dit-kubernetes-*" names,
	// or reinstalling a custom-named context leaves the stale containers behind.
	for _, want := range []string{"dit-ctx-server", "dit-ctx-launch"} {
		found := false
		for _, got := range d.RemovedNames {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Remove(%q); removed names were %v", want, d.RemovedNames)
		}
	}
	if !strings.Contains(output, "Initializing dit infrastructure") {
		t.Errorf("expected initial banner, got %q", output)
	}
}

func TestK8sInstall_LaunchFailurePanics(t *testing.T) {
	d := &fakeDocker{
		latestDownloaded: true,
		launchK8sErr:     errors.New("docker daemon down"),
		launchK8sOut:     "boom",
	}
	panicked := false
	_ = captureStdout(func() {
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
				}
			}()
			_, _ = captureExit(t, func() {
				with(t, d, &fakeK8s{}, func() {
					Install("v1.0.0", "dit", false, 9999, "ctx", nil)
				})
			})
		}()
	})
	if !panicked {
		t.Errorf("expected panic on LaunchDitKubernetesServers failure")
	}
}

func TestK8sInstall_TagWarningsAreNonFatal(t *testing.T) {
	d := &fakeDocker{
		latestDownloaded: false,
		tagErr:           errors.New("tag failed"),
	}
	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Install("v1.0.0", "dit", false, 9999, "ctx", nil)
			})
		})
	})

	if !strings.Contains(output, "Error tagging image") {
		t.Errorf("expected tag warning, got %q", output)
	}
}

func TestK8sInstall_PropertiesExportContextConfig(t *testing.T) {
	// t.Setenv isolates DIT_CONTEXT_CONFIG and restores it after the test.
	t.Setenv("DIT_CONTEXT_CONFIG", "")
	d := &fakeDocker{
		latestDownloaded: true,
	}
	_ = captureStdout(func() {
		_, _ = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Install("v1.0.0", "dit", false, 9999, "ctx",
					[]string{"storageClass=csi-hostpath-sc", "snapshotClass=csi-hostpath-snapclass"})
			})
		})
	})

	got := os.Getenv("DIT_CONTEXT_CONFIG")
	want := "storageClass=csi-hostpath-sc,snapshotClass=csi-hostpath-snapclass"
	if got != want {
		t.Errorf("DIT_CONTEXT_CONFIG = %q, want %q", got, want)
	}
}

func TestK8sInstall_InvalidPropertyExits(t *testing.T) {
	d := &fakeDocker{latestDownloaded: true}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Install("v1.0.0", "dit", false, 9999, "ctx", []string{"storageClassMissingEquals"})
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "invalid context parameter") {
		t.Errorf("expected invalid param error, got %q", output)
	}
}
