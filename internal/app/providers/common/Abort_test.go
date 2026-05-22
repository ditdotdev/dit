package common

import (
	"net/http"
	"strings"
	"testing"
)

func TestAbort_NoRunningOperations(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			Abort("repo", port)
		})
	})

	if !didExit {
		t.Errorf("Abort with no running operations should osExit(0)")
	}
	if code != 0 {
		t.Errorf("Abort exit code = %d, want 0", code)
	}
	if !strings.Contains(output, "no operation in progress") {
		t.Errorf("expected 'no operation in progress' in output, got %q", output)
	}
}

func TestAbort_AbortsRunningOperations(t *testing.T) {
	abortCalled := false
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/operations"):
			_, _ = w.Write([]byte(`[{"id":"op1","type":"PUSH","state":"RUNNING","remote":"origin","commitId":"c"}]`))
		case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/operations/op1"):
			abortCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			Abort("repo", port)
		})
	})

	if didExit {
		t.Errorf("Abort with running op should not exit")
	}
	if !abortCalled {
		t.Errorf("expected AbortOperation API call")
	}
	if !strings.Contains(output, "aborting operation op1") {
		t.Errorf("expected aborting message, got %q", output)
	}
}

func TestAbort_AbortAPIErrorIsWarning(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/operations"):
			_, _ = w.Write([]byte(`[{"id":"op1","type":"PUSH","state":"RUNNING","remote":"origin","commitId":"c"}]`))
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/operations/op1"):
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":"X","message":"server error","details":""}`))
		}
	})

	output := captureStdout(func() {
		_, _ = captureExit(t, func() { Abort("repo", port) })
	})

	if !strings.Contains(output, "Warning: Failed to abort operation op1") {
		t.Errorf("expected warning about failed abort, got %q", output)
	}
}

func TestAbort_SkipsNonRunningOperations(t *testing.T) {
	abortCalled := false
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/operations"):
			_, _ = w.Write([]byte(`[{"id":"op1","type":"PUSH","state":"COMPLETE","remote":"origin","commitId":"c"}]`))
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/operations/op1"):
			abortCalled = true
			w.WriteHeader(http.StatusOK)
		}
	})

	var didExit bool
	var code int
	_ = captureStdout(func() {
		didExit, code = captureExit(t, func() { Abort("repo", port) })
	})

	if abortCalled {
		t.Errorf("Abort should not call AbortOperation on a COMPLETE operation")
	}
	if !didExit || code != 0 {
		t.Errorf("Abort with only non-running ops should osExit(0); got didExit=%v code=%d", didExit, code)
	}
}
