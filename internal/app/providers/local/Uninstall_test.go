package local

import (
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
		volumeExists:              map[string]bool{"dit-ctx-data": true},
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
