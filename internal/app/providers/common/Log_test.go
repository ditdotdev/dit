package common

import (
	"net/http"
	"strings"
	"testing"
)

func TestLog_ListError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":"X","message":"boom","details":""}`))
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			Log("repo", nil, port)
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on list error; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Error listing commits for repo") {
		t.Errorf("expected error message, got %q", output)
	}
}

func TestLog_PrintsCommits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"c1","properties":{"author":"alice","user":"alice","email":"a@x","timestamp":"2025-01-01","message":"first","tags":{"v1":"","env":"prod"}}},
			{"id":"c2","properties":{"author":"bob","message":"second"}}
		]`))
	})

	output := captureStdout(func() {
		_, _ = captureExit(t, func() { Log("repo", nil, port) })
	})

	for _, want := range []string{"commit c1", "alice", "Tags:", "env=prod", "first", "commit c2", "bob", "second"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got %q", want, output)
		}
	}
}

func TestLog_TagsWithoutValue(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"c1","properties":{"tags":{"prod":""}}}
		]`))
	})

	output := captureStdout(func() {
		_, _ = captureExit(t, func() { Log("repo", nil, port) })
	})

	if !strings.Contains(output, "Tags:") || !strings.Contains(output, "prod") {
		t.Errorf("expected 'Tags: prod' in output, got %q", output)
	}
}
