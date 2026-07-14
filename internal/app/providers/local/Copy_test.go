// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestCopy_ContainerInspectError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{inspectContainerErr: errors.New("no such container")}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Copy("repo", "", "/local/src", "/data", port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Container information is not available") {
		t.Errorf("expected container info error, got %q", output)
	}
}

func TestCopy_EmptyContainerInfoExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{inspectContainerOut: ""}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Copy("repo", "", "/local/src", "/data", port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Container information is not available") {
		t.Errorf("expected container info error, got %q", output)
	}
}

func TestCopy_MountsParseFailure(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		getValFromContainer: map[string]string{"repo:Mounts": "not-json"},
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Copy("repo", "", "/local/src", "", port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Failed to unmarshal mounts") {
		t.Errorf("expected unmarshal error, got %q", output)
	}
}

func TestCopy_HappyPath(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/volumes/v0/activate"):
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "/volumes/v0/deactivate"):
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/volumes/v0"):
			_, _ = w.Write([]byte(`{"name":"v0","properties":{"path":"/data"},"config":{"mountpoint":"/var/lib/zfs/v0"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		getValFromContainer: map[string]string{
			"repo:Mounts":        `[{"Type":"volume","Name":"repo_v0","Source":"/var/lib/docker/volumes/repo_v0","Target":"/data","Destination":"/data"}]`,
			"repo:State.Running": "false",
		},
	}

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Copy("repo", "", "/local/src", "", port, "ctx")
			})
		})
	})

	if didExit {
		t.Errorf("happy path should not exit; output=%q", output)
	}
	if !strings.Contains(output, "running with data from /local/src") {
		t.Errorf("expected final running message, got %q", output)
	}
}

func TestCopy_MultiMountRequiresPath(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		getValFromContainer: map[string]string{
			"repo:Mounts": `[{"Destination":"/a"},{"Destination":"/b"}]`,
		},
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Copy("repo", "", "/local/src", "", port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on multi-mount no-path; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "--destination is required") {
		t.Errorf("expected destination-required error, got %q", output)
	}
}

func TestCopy_RunningContainerStopsAndStarts(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/volumes/v0"):
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"name":"v0","properties":{"path":"/data"},"config":{"mountpoint":"/mnt"}}`))
			} else {
				w.WriteHeader(http.StatusOK)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		getValFromContainer: map[string]string{
			"repo:Mounts":        `[{"Type":"volume","Name":"repo_v0","Destination":"/data"}]`,
			"repo:State.Running": "true",
		},
	}

	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Copy("repo", "", "/local/src", "", port, "ctx")
			})
		})
	})

	if d.StopCalls != 1 {
		t.Errorf("expected Stop call for running container, got %d", d.StopCalls)
	}
	if d.StartCalls != 1 {
		t.Errorf("expected Start call to restore container, got %d", d.StartCalls)
	}
	if !strings.Contains(output, "running with data from") {
		t.Errorf("expected restart message, got %q", output)
	}
}
