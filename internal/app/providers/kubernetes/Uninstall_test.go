package kubernetes

import (
	"net/http"
	"strings"
	"testing"
)

func TestK8sUninstall_NothingFound(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	d := &fakeDocker{}
	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Uninstall(false, false, "ctx", port)
			})
		})
	})

	if !strings.Contains(output, "nothing to uninstall") {
		t.Errorf("expected nothing-found message, got %q", output)
	}
}

func TestK8sUninstall_ReposExistRequireForce(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"r1","properties":{}}]`))
	})

	d := &fakeDocker{serverAvailable: true}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Uninstall(false, false, "ctx", port)
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

func TestK8sUninstall_FullFlow(t *testing.T) {
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
		serverAvailable: true,
		volumeExists:    map[string]bool{"datadatdat-ctx-data": true},
	}
	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Uninstall(true, true, "ctx", port)
			})
		})
	})

	if !strings.Contains(output, "Uninstalled datadatdat infrastructure") {
		t.Errorf("expected uninstalled summary, got %q", output)
	}
	if !strings.Contains(output, "server container") || !strings.Contains(output, "data volume") || !strings.Contains(output, "images") {
		t.Errorf("expected each removed item in summary, got %q", output)
	}
}
