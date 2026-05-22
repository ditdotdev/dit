package common

import (
	"net/http"
	"strings"
	"testing"
)

func TestDeleteCommit_Success(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	output := captureStdout(func() {
		DeleteCommit("repo", "abc", port)
	})

	if !strings.Contains(output, "abc deleted") {
		t.Errorf("expected 'abc deleted', got %q", output)
	}
}

func TestDeleteCommit_Error(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NotFound","message":"missing","details":""}`))
	})

	output := captureStdout(func() {
		DeleteCommit("repo", "abc", port)
	})

	if !strings.Contains(output, "Error deleting commit abc") {
		t.Errorf("expected error message, got %q", output)
	}
}

func TestDeleteTags_RemovesMatchingTags(t *testing.T) {
	var receivedCommit string
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"abc","properties":{"tags":{"v1":"","env":"prod"}}}`))
		case http.MethodPost:
			// Capture the updated commit body for assertion
			receivedCommit = r.URL.Path
			_, _ = w.Write([]byte(`{"id":"abc","properties":{"tags":{}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	_ = captureStdout(func() {
		DeleteTags("repo", "abc", []string{"v1"}, port)
	})

	if !strings.Contains(receivedCommit, "/commits/abc") {
		t.Errorf("expected POST to commit endpoint, got %q", receivedCommit)
	}
}

func TestDeleteTags_TagWithValue(t *testing.T) {
	called := false
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"abc","properties":{"tags":{"env":"prod"}}}`))
		case http.MethodPost:
			called = true
			_, _ = w.Write([]byte(`{"id":"abc","properties":{}}`))
		}
	})

	_ = captureStdout(func() {
		DeleteTags("repo", "abc", []string{"env=prod"}, port)
	})

	if !called {
		t.Errorf("expected UpdateCommit to be called")
	}
}

func TestDeleteTags_UpdateError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"abc","properties":{"tags":{"v1":""}}}`))
		case http.MethodPost:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":"X","message":"boom","details":""}`))
		}
	})

	output := captureStdout(func() {
		DeleteTags("repo", "abc", []string{"v1"}, port)
	})

	if !strings.Contains(output, "Error updating commit tags") {
		t.Errorf("expected error message, got %q", output)
	}
}
