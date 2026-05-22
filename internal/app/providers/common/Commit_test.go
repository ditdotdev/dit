package common

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCommit_RepoNotFound(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NotFound","message":"missing","details":""}`))
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			Commit("missing", "msg", nil, "user", "user@x", port)
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on missing repo; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "repository 'missing' not found") {
		t.Errorf("expected not-found message, got %q", output)
	}
}

func TestCommit_HappyPath(t *testing.T) {
	var createdCommit string
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/repositories/repo"):
			_, _ = w.Write([]byte(`{"name":"repo","properties":{}}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/repositories/repo/status"):
			_, _ = w.Write([]byte(`{"lastCommit":"prev","sourceCommit":"prev"}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/repo/commits"):
			body, _ := io.ReadAll(r.Body)
			var c struct {
				Id string `json:"id"`
			}
			_ = json.Unmarshal(body, &c)
			createdCommit = c.Id
			_, _ = w.Write([]byte(`{"id":"` + c.Id + `","properties":{}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			Commit("repo", "msg", []string{"v1", "env=prod"}, "user", "user@x", port)
		})
	})

	if didExit {
		t.Errorf("Commit happy path should not exit")
	}
	if createdCommit == "" {
		t.Errorf("expected commit creation")
	}
	if !strings.Contains(output, "Commit "+createdCommit) {
		t.Errorf("expected 'Commit <id>' in output, got %q", output)
	}
}

func TestCommit_CreateCommitError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/repositories/repo"):
			_, _ = w.Write([]byte(`{"name":"repo","properties":{}}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/status"):
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"code":"Conflict","message":"commit exists","details":""}`))
		}
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			Commit("repo", "msg", nil, "user", "user@x", port)
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on create error; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "commit exists") {
		t.Errorf("expected server message in output, got %q", output)
	}
}
