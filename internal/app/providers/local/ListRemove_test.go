package local

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestList_PrintsReposWithStatus(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"r1","properties":{}},{"name":"r2","properties":{}}]`))
	})

	d := &fakeDocker{
		getValFromContainer: map[string]string{
			"r1:State.Status": "running",
		},
		getValFromContainerErrs: map[string]error{
			"r2:State.Status": errors.New("no such container"),
		},
	}

	output := captureStdout(func() {
		withDocker(t, d, func() {
			List("docker", port)
		})
	})

	if !strings.Contains(output, "r1") || !strings.Contains(output, "running") {
		t.Errorf("expected r1 + running, got %q", output)
	}
	if !strings.Contains(output, "r2") || !strings.Contains(output, "detached") {
		t.Errorf("expected r2 + detached fallback, got %q", output)
	}
}

func TestRemove_HappyPath(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/volumes") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"name":"v0","properties":{"path":"/data"}}]`))
		case strings.Contains(r.URL.Path, "/volumes/v0/deactivate"):
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/repositories/repo") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	d := &fakeDocker{
		getValFromContainer: map[string]string{
			"repo:Id":           "abc123",
			"repo:State.Status": "stopped",
		},
		containerIsRunning: map[string]bool{"repo": false},
	}

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Remove("repo", false, port, "ctx")
			})
		})
	})

	if didExit {
		t.Errorf("happy path should not exit; output=%q", output)
	}
	if !strings.Contains(output, "repo removed") {
		t.Errorf("expected removed message, got %q", output)
	}
}

func TestRemove_RunningContainerRequiresForce(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	d := &fakeDocker{
		getValFromContainer: map[string]string{
			"repo:Id":           "abc",
			"repo:State.Status": "running",
		},
	}

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Remove("repo", false, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on running+no-force; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "is running, stop or use '-f' to force") {
		t.Errorf("expected running warning, got %q", output)
	}
}

func TestRemove_ForceRunningContainer(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/volumes") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	d := &fakeDocker{
		getValFromContainer: map[string]string{
			"repo:Id":           "abc",
			"repo:State.Status": "running",
		},
		containerIsRunning: map[string]bool{"repo": true},
	}

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Remove("repo", true, port, "ctx")
			})
		})
	})

	if didExit {
		t.Errorf("force should bypass running check; output=%q", output)
	}
	if d.RemoveCalls != 1 {
		t.Errorf("expected docker.Remove call, got %d", d.RemoveCalls)
	}
}

func TestRemove_NonExistentRepoExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/volumes") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[]`))
		case strings.HasSuffix(r.URL.Path, "/repositories/repo") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NotFound","message":"x","details":""}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	d := &fakeDocker{}

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Remove("repo", true, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on missing repo; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "does not exist") {
		t.Errorf("expected does-not-exist error, got %q", output)
	}
}

func TestRemove_RemoveVolumeFallsBackToAPI(t *testing.T) {
	apiDeleteCalled := false
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/volumes") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"name":"v0","properties":{"path":"/data"}}]`))
		case strings.Contains(r.URL.Path, "/volumes/v0/deactivate"):
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/volumes/v0") && r.Method == http.MethodDelete:
			apiDeleteCalled = true
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/repositories/repo") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	d := &fakeDocker{
		getValFromContainer: map[string]string{
			"repo:Id": "",
		},
		removeVolumeErr: errors.New("docker volume rm failed"),
	}

	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Remove("repo", true, port, "ctx")
			})
		})
	})

	if !apiDeleteCalled {
		t.Errorf("expected fallback to volumesApi.DeleteVolume; output=%q", output)
	}
}
