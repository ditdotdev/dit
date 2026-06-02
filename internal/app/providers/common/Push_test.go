package common

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// pushServerHandler handles the API surface Push uses on the happy path.
// Tests can override individual responses via the opts struct.
func pushServerHandler(t *testing.T, apiBase string, opts pushOpts) http.HandlerFunc {
	t.Helper()
	remoteJSON := `{"provider":"dit","name":"origin","properties":{"host":"example.com","org":"o","repo":"r","api_base_url":"` + apiBase + `"}}`
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/remotes"):
			if opts.listRemotesStatus != 0 {
				w.WriteHeader(opts.listRemotesStatus)
				_, _ = w.Write([]byte(`{"code":"X","message":"err","details":""}`))
				return
			}
			_, _ = w.Write([]byte(`[` + remoteJSON + `]`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/repositories/repo/status"):
			if opts.noLastCommit {
				_, _ = w.Write([]byte(`{}`))
				return
			}
			_, _ = w.Write([]byte(`{"lastCommit":"c1","sourceCommit":"c1"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/remotes/origin"):
			if opts.getRemoteStatus != 0 {
				w.WriteHeader(opts.getRemoteStatus)
				_, _ = w.Write([]byte(`{"code":"X","message":"err","details":""}`))
				return
			}
			_, _ = w.Write([]byte(remoteJSON))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/push"):
			if opts.pushStatus != 0 {
				w.WriteHeader(opts.pushStatus)
				_, _ = w.Write([]byte(`{"code":"X","message":"push-err","details":""}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"op1","type":"PUSH","state":"RUNNING","remote":"origin","commitId":"c1"}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/remotes/origin/commits"):
			if opts.duplicateRemoteCommit {
				_, _ = w.Write([]byte(`[{"id":"c1","properties":{}}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/repositories/repo/commits/c1"):
			_, _ = w.Write([]byte(`{"id":"c1","properties":{}}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/repositories/repo/commits"):
			if opts.emptyTagCommits {
				_, _ = w.Write([]byte(`[]`))
				return
			}
			_, _ = w.Write([]byte(`[{"id":"c1","properties":{}}]`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/operations/op1/progress"):
			_, _ = w.Write([]byte(`[{"id":1,"type":"COMPLETE"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

type pushOpts struct {
	listRemotesStatus     int
	getRemoteStatus       int
	pushStatus            int
	noLastCommit          bool
	duplicateRemoteCommit bool
	emptyTagCommits       bool
}

// startRemoteAPIServer is a separate httptest server that stands in for the
// remote provider's api_base_url (used by ensureRemoteRepoExists).
func startRemoteAPIServer(t *testing.T, statusCode int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestPush_ListRemotesError(t *testing.T) {
	apiBase := startRemoteAPIServer(t, http.StatusOK)
	port := startMockServer(t, pushServerHandler(t, apiBase, pushOpts{listRemotesStatus: http.StatusInternalServerError}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Push("repo", "", "", nil, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "remote is not set") {
		t.Errorf("expected error msg, got %q", output)
	}
}

func TestPush_NoLastCommit(t *testing.T) {
	apiBase := startRemoteAPIServer(t, http.StatusOK)
	port := startMockServer(t, pushServerHandler(t, apiBase, pushOpts{noLastCommit: true}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Push("repo", "", "", nil, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "container has no history") {
		t.Errorf("expected error msg, got %q", output)
	}
}

func TestPush_GetRemoteError(t *testing.T) {
	apiBase := startRemoteAPIServer(t, http.StatusOK)
	port := startMockServer(t, pushServerHandler(t, apiBase, pushOpts{getRemoteStatus: http.StatusNotFound}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Push("repo", "", "", nil, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "remote 'origin' not found") {
		t.Errorf("expected error msg, got %q", output)
	}
}

func TestPush_TagsWithCommitExits(t *testing.T) {
	apiBase := startRemoteAPIServer(t, http.StatusOK)
	port := startMockServer(t, pushServerHandler(t, apiBase, pushOpts{}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Push("repo", "guid", "", []string{"v1"}, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "tags cannot be specified when commit") {
		t.Errorf("expected error msg, got %q", output)
	}
}

func TestPush_HappyPath(t *testing.T) {
	apiBase := startRemoteAPIServer(t, http.StatusOK)
	port := startMockServer(t, pushServerHandler(t, apiBase, pushOpts{}))

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() { Push("repo", "", "", nil, false, port) })
	})

	if didExit {
		t.Errorf("happy path should not exit; output=%q", output)
	}
	if !strings.Contains(output, "Push completed successfully") {
		t.Errorf("expected success msg, got %q", output)
	}
}

func TestPush_DuplicateCommitExits(t *testing.T) {
	apiBase := startRemoteAPIServer(t, http.StatusOK)
	port := startMockServer(t, pushServerHandler(t, apiBase, pushOpts{duplicateRemoteCommit: true}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Push("repo", "", "", nil, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on duplicate; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "exists in remote") {
		t.Errorf("expected dedup msg, got %q", output)
	}
}

func TestPush_MetadataOnlySkipsDedupe(t *testing.T) {
	apiBase := startRemoteAPIServer(t, http.StatusOK)
	port := startMockServer(t, pushServerHandler(t, apiBase, pushOpts{duplicateRemoteCommit: true}))

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() { Push("repo", "", "", nil, true, port) })
	})

	if didExit {
		t.Errorf("metadata-only should bypass dedupe; output=%q", output)
	}
	if !strings.Contains(output, "Push completed successfully") {
		t.Errorf("expected success msg, got %q", output)
	}
}

func TestPush_NoMatchingCommitsExits(t *testing.T) {
	apiBase := startRemoteAPIServer(t, http.StatusOK)
	port := startMockServer(t, pushServerHandler(t, apiBase, pushOpts{emptyTagCommits: true}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Push("repo", "", "", []string{"missing"}, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "no matching commits found") {
		t.Errorf("expected error msg, got %q", output)
	}
}

func TestPush_PushOperationError(t *testing.T) {
	apiBase := startRemoteAPIServer(t, http.StatusOK)
	port := startMockServer(t, pushServerHandler(t, apiBase, pushOpts{pushStatus: http.StatusInternalServerError}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Push("repo", "", "", nil, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "push-err") {
		t.Errorf("expected server error msg, got %q", output)
	}
}

func TestPush_EnsureRemoteRepoFails(t *testing.T) {
	apiBase := startRemoteAPIServer(t, http.StatusInternalServerError)
	port := startMockServer(t, pushServerHandler(t, apiBase, pushOpts{}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Push("repo", "", "", nil, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "failed to create remote repository") {
		t.Errorf("expected ensureRemoteRepoExists error msg, got %q", output)
	}
}

func TestEnsureRemoteRepoExists_MissingFields(t *testing.T) {
	err := ensureRemoteRepoExists(map[string]interface{}{})
	if err == nil {
		t.Errorf("expected error when fields missing")
	}
}

func TestEnsureRemoteRepoExists_AlreadyExists(t *testing.T) {
	apiBase := startRemoteAPIServer(t, http.StatusOK)
	err := ensureRemoteRepoExists(map[string]interface{}{
		"api_base_url": apiBase,
		"org":          "o",
		"repo":         "r",
	})
	if err != nil {
		t.Errorf("expected no error when repo exists, got %v", err)
	}
}

func TestEnsureRemoteRepoExists_Creates(t *testing.T) {
	createCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			createCalled = true
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	err := ensureRemoteRepoExists(map[string]interface{}{
		"api_base_url": srv.URL,
		"org":          "o",
		"repo":         "r",
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !createCalled {
		t.Errorf("expected POST to create repo")
	}
}

func TestEnsureRemoteRepoExists_ServerErrorOnCreate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	err := ensureRemoteRepoExists(map[string]interface{}{
		"api_base_url": srv.URL,
		"org":          "o",
		"repo":         "r",
	})
	if err == nil {
		t.Errorf("expected error on create failure")
	}
}
