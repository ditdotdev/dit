// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"net/http"
	"strings"
	"testing"
)

// ditRemoteJSON references the dit provider, one of several
// registered in providers/common via blank-imports in RemoteAdd.go (dit,
// ssh, s3, s3web, nop).
const ditRemoteJSON = `{"provider":"dit","name":"origin","properties":{"host":"example.com","org":"o","repo":"r"}}`

func TestRemoteList_Success(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[` + ditRemoteJSON + `]`))
	})

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() { RemoteList("repo", port) })
	})

	if didExit {
		t.Errorf("RemoteList should not exit on success")
	}
	for _, want := range []string{"REMOTE", "URI", "origin"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got %q", want, output)
		}
	}
}

func TestRemoteList_UnknownProviderExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"provider":"nonsense","name":"origin","properties":{}}]`))
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { RemoteList("repo", port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on unknown provider; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "nonsense") {
		t.Errorf("expected provider name in error, got %q", output)
	}
}

func TestRemoteRemove_Success(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	output := captureStdout(func() {
		RemoteRemove("repo", "origin", port)
	})

	if !strings.Contains(output, "Removed origin from repo") {
		t.Errorf("expected removed message, got %q", output)
	}
}

func TestRemoteRemove_Error(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NotFound","message":"missing","details":""}`))
	})

	output := captureStdout(func() {
		RemoteRemove("repo", "origin", port)
	})

	if !strings.Contains(output, "Error removing remote origin") {
		t.Errorf("expected error message, got %q", output)
	}
}

func TestRemoteLog_NoRemotesExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { RemoteLog("repo", "origin", nil, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) when no remotes; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "remote is not set") {
		t.Errorf("expected 'remote is not set', got %q", output)
	}
}

func TestRemoteLog_UnknownProviderExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"provider":"nonsense","name":"origin","properties":{}}]`))
	})

	var didExit bool
	var code int
	_ = captureStdout(func() {
		didExit, code = captureExit(t, func() { RemoteLog("repo", "origin", nil, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on unknown provider; got didExit=%v code=%d", didExit, code)
	}
}

func TestRemoteLog_ListRemoteCommitsError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/remotes"):
			_, _ = w.Write([]byte(`[` + ditRemoteJSON + `]`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":"X","message":"err","details":""}`))
		}
	})

	output := captureStdout(func() {
		_, _ = captureExit(t, func() { RemoteLog("repo", "origin", nil, port) })
	})

	if !strings.Contains(output, "origin has not been initialized") {
		t.Errorf("expected initialization warning, got %q", output)
	}
}

func TestRemoteLog_PrintsCommits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/remotes"):
			_, _ = w.Write([]byte(`[` + ditRemoteJSON + `]`))
		case strings.Contains(r.URL.Path, "/commits"):
			_, _ = w.Write([]byte(`[
				{"id":"c1","properties":{"author":"alice","tags":{"v1":"","env":"prod"},"message":"first"}},
				{"id":"c2","properties":{}}
			]`))
		}
	})

	output := captureStdout(func() {
		_, _ = captureExit(t, func() { RemoteLog("repo", "origin", nil, port) })
	})

	for _, want := range []string{"Commit c1", "alice", "Tags:", "env=prod", "first", "Commit c2"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got %q", want, output)
		}
	}
}

func TestRemoteAdd_AlreadyExistsExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(ditRemoteJSON))
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			RemoteAdd("repo", "nop://x", "", nil, port)
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) when remote exists; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "remote origin already exists") {
		t.Errorf("expected already-exists message, got %q", output)
	}
}

func TestRemoteAdd_ParseURIFailureExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NotFound","message":"missing","details":""}`))
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			RemoteAdd("repo", "totally bogus uri", "", nil, port)
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on parse failure; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Error parsing URI") {
		t.Errorf("expected parse error, got %q", output)
	}
}

func TestRemoteAdd_HappyPath(t *testing.T) {
	createdRemote := false
	updatedRepo := false
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/remotes/origin"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NotFound","message":"x","details":""}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/remotes"):
			createdRemote = true
			_, _ = w.Write([]byte(ditRemoteJSON))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/repositories/repo"):
			_, _ = w.Write([]byte(`{"name":"repo","properties":{}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/repo"):
			updatedRepo = true
			_, _ = w.Write([]byte(`{"name":"repo","properties":{"remote":"origin"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	var didExit bool
	_ = captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			RemoteAdd("repo", "http://example.com/o/r", "", nil, port)
		})
	})

	if didExit {
		t.Errorf("RemoteAdd happy path should not exit")
	}
	if !createdRemote {
		t.Errorf("expected CreateRemote call")
	}
	if !updatedRepo {
		t.Errorf("expected UpdateRepository call")
	}
}

func TestRemoteAdd_CreateRemoteErrorExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/remotes/origin"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NotFound","message":"x","details":""}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/remotes"):
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":"X","message":"boom","details":""}`))
		}
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			RemoteAdd("repo", "http://example.com/o/r", "", nil, port)
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on create failure; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Error creating remote") {
		t.Errorf("expected error message, got %q", output)
	}
}

func TestRemoteAdd_CustomRemoteName(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/remotes/upstream"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NotFound","message":"x","details":""}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/remotes"):
			_, _ = w.Write([]byte(`{"provider":"dit","name":"upstream","properties":{"host":"example.com","org":"o","repo":"r"}}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/repositories/repo"):
			_, _ = w.Write([]byte(`{"name":"repo","properties":{"existing":"prop"}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/repo"):
			_, _ = w.Write([]byte(`{"name":"repo","properties":{}}`))
		}
	})

	var didExit bool
	_ = captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			RemoteAdd("repo", "http://example.com/o/r", "upstream", nil, port)
		})
	})

	if didExit {
		t.Errorf("RemoteAdd with custom name should not exit on happy path")
	}
}
