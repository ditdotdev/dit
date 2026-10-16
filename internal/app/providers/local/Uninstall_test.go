// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestUninstall_NothingFound(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})

	d := &fakeDocker{}
	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Uninstall("v1.0.0", false, false, port, "ctx")
			})
		})
	})

	if !strings.Contains(output, "nothing to uninstall") {
		t.Errorf("expected nothing-found message, got %q", output)
	}
}

func TestUninstall_ReposExistRequiresForce(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"r1","properties":{}}]`))
	})

	d := &fakeDocker{ditServerAvailable: true}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Uninstall("v1.0.0", false, false, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "repository 'r1' exists") {
		t.Errorf("expected repos-exist message, got %q", output)
	}
}

func TestUninstall_FullFlow(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/repositories":
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	d := &fakeDocker{
		ditServerAvailable: true,
		ditLaunchAvailable: true,
		volumeExists:       map[string]bool{"dit-ctx-data": true},
	}

	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Uninstall("v1.0.0", true, true, port, "ctx")
			})
		})
	})

	if !strings.Contains(output, "Uninstalled dit infrastructure") {
		t.Errorf("expected uninstalled summary, got %q", output)
	}
	if !strings.Contains(output, "server container") {
		t.Errorf("expected server container removal, got %q", output)
	}
	if !strings.Contains(output, "launch container") {
		t.Errorf("expected launch container removal, got %q", output)
	}
	if !strings.Contains(output, "data volume") {
		t.Errorf("expected data volume removal, got %q", output)
	}
	if !strings.Contains(output, "images") {
		t.Errorf("expected images removal, got %q", output)
	}
}

// #244: when the ZFS teardown container times out (the docker daemon
// wedged on its cleanup), uninstall must warn, show whatever the
// teardown script already printed, and still remove the volume and
// images rather than hang or abort.
func TestUninstall_TeardownTimeoutIsWarningAndProceeds(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})

	d := &fakeDocker{
		ditServerAvailable: true,
		ditLaunchAvailable: true,
		teardownErr: fmt.Errorf("teardown container dit-ctx-teardown did not exit within 5m0s; forcibly removed it: %w",
			context.DeadlineExceeded),
		teardownOut:  "Destroying pool dit-ctx\nRemoving docker plugin socket\n",
		volumeExists: map[string]bool{"dit-ctx-data": true},
	}

	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Uninstall("v1.0.0", true, true, port, "ctx")
			})
		})
	})

	if !strings.Contains(output, "Teardown encountered errors") {
		t.Errorf("expected teardown warning, got %q", output)
	}
	if !strings.Contains(output, "did not exit within") {
		t.Errorf("expected the timeout cause in the warning, got %q", output)
	}
	if !strings.Contains(output, "Removing docker plugin socket") {
		t.Errorf("expected the teardown script's captured output, got %q", output)
	}
	if !strings.Contains(output, "Uninstalled dit infrastructure") {
		t.Errorf("expected uninstall to finish, got %q", output)
	}
	for _, want := range []string{"server container", "launch container", "data volume", "images"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in the summary (uninstall must continue past a stuck teardown), got %q", want, output)
		}
	}
}
