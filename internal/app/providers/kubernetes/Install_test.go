package kubernetes

import (
	"errors"
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
				Install("v1.0.0", "datadatdat", false, 9999, "ctx")
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
				Install("v1.0.0", "datadatdat", false, 9999, "ctx")
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
				Install("v1.0.0", "datadatdat", false, 9999, "ctx")
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
		fetchLaunchLogs:  []string{"DATADATDAT START 2024-01-01 00:00:00 hello", "verbose-line", "DATADATDAT END", "DATADATDAT FINISHED"},
	}

	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Install("v1.0.0", "datadatdat", true, 9999, "ctx")
			})
		})
	})

	if d.RemoveCalls < 2 {
		t.Errorf("expected docker.Remove calls for stale server + launch, got %d", d.RemoveCalls)
	}
	if !strings.Contains(output, "Initializing datadatdat infrastructure") {
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
					Install("v1.0.0", "datadatdat", false, 9999, "ctx")
				})
			})
		}()
	})
	if !panicked {
		t.Errorf("expected panic on LaunchDatadatdatKubernetesServers failure")
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
				Install("v1.0.0", "datadatdat", false, 9999, "ctx")
			})
		})
	})

	if !strings.Contains(output, "Error tagging image") {
		t.Errorf("expected tag warning, got %q", output)
	}
}
