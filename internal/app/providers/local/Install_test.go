// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"errors"
	"strings"
	"testing"
)

func TestInstall_DockerNotAvailableExits(t *testing.T) {
	d := &fakeDocker{versionErr: errors.New("no docker")}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Install("v1.0.0", "dit", false, 9999, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Error checking docker version") {
		t.Errorf("expected docker-error message, got %q", output)
	}
}

func TestInstall_LatestNotDownloadedTriggersPull(t *testing.T) {
	d := &fakeDocker{
		ditLatestIsDownloaded: false,
		fetchLaunchLogs:       []string{"DIT START 2024-01-01 00:00:00 hello", "DIT FINISHED"},
		launchOut:             "ok",
	}
	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Install("v1.0.0", "dit", true, 9999, "ctx")
			})
		})
	})

	if d.PullCalls < 1 {
		t.Errorf("expected Pull call; pull calls=%d output=%q", d.PullCalls, output)
	}
}

func TestInstall_LatestNotDownloadedLocalFallback(t *testing.T) {
	d := &fakeDocker{
		ditLatestIsDownloaded: false,
		fetchLaunchLogs:       []string{"DIT FINISHED"},
		launchOut:             "ok",
	}
	_ = captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Install("v1.0.0", "local", false, 9999, "ctx")
			})
		})
	})

	if d.PullCalls < 1 {
		t.Errorf("expected Pull from local fallback to dit, got %d", d.PullCalls)
	}
}

func TestInstall_EchoesLaunchErrorLines(t *testing.T) {
	d := &fakeDocker{
		ditLatestIsDownloaded: true,
		fetchLaunchLogs: []string{
			"DIT ERROR 2024-01-01 00:00:00 Failed to load ZFS",
			"DIT FINISHED",
		},
		launchOut: "ok",
	}
	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Install("v1.0.0", "dit", false, 9999, "ctx")
			})
		})
	})

	if !strings.Contains(output, "Error: Failed to load ZFS") {
		t.Errorf("expected launch ERROR line echoed, got %q", output)
	}
}

func TestInstall_RemovesPreexistingServer(t *testing.T) {
	d := &fakeDocker{
		ditLatestIsDownloaded: true,
		ditServerAvailable:    true,
		ditLaunchAvailable:    true,
		fetchLaunchLogs:       []string{"DIT FINISHED"},
		launchOut:             "ok",
	}
	_ = captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Install("v1.0.0", "dit", false, 9999, "ctx")
			})
		})
	})

	if d.RemoveCalls < 2 {
		t.Errorf("expected docker.Remove calls for server + launch, got %d", d.RemoveCalls)
	}
}

func TestInstall_LaunchFailurePanics(t *testing.T) {
	d := &fakeDocker{
		ditLatestIsDownloaded: true,
		launchErr:             errors.New("docker daemon down"),
		launchOut:             "boom",
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
				withDocker(t, d, func() {
					Install("v1.0.0", "dit", false, 9999, "ctx")
				})
			})
		}()
	})
	if !panicked {
		t.Errorf("expected panic on LaunchDitServers failure")
	}
}
