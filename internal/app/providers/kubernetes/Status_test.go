package kubernetes

import (
	"net/http"
	"strings"
	"testing"
)

func TestStatus_RepoNotFound(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NotFound","message":"missing","details":""}`))
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, &fakeDocker{}, &fakeK8s{}, func() {
				Status("r1", port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "repository 'r1' not found") {
		t.Errorf("expected not-found message, got %q", output)
	}
}

func TestStatus_FullOutput(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/repositories/r1/status":
			_, _ = w.Write([]byte(`{"lastCommit":"abc","sourceCommit":"abc"}`))
		case "/v1/repositories/r1/volumes":
			_, _ = w.Write([]byte(`[{"name":"v0","properties":{"path":"/data"}}]`))
		default:
			if strings.HasSuffix(r.URL.Path, "/volumes/v0/status") {
				_, _ = w.Write([]byte(`{"name":"v0","logicalSize":2048,"actualSize":1024,"properties":{"path":"/data"},"ready":true}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
	})

	k := &fakeK8s{statefulSetStatus: "running"}
	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			with(t, &fakeDocker{}, k, func() {
				Status("r1", port, "ctx")
			})
		})
	})

	for _, want := range []string{"Status:", "running", "Last Commit:", "abc", "Source Commit:", "Volume", "/data", "2.0 KiB"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got %q", want, output)
		}
	}
}
