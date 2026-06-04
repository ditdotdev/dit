package common

import (
	"net/http"
	"strings"
	"testing"
)

const ditRemoteWithAPI = `{"provider":"dit","name":"origin","properties":{"host":"example.com","org":"o","repo":"r","api_base_url":"http://api"}}`

// pullServerHandler returns a handler covering the API surface Pull uses on
// the happy path. Tests can override individual paths by composing.
func pullServerHandler(t *testing.T, opts pullOpts) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/remotes"):
			if opts.listRemotesStatus != 0 {
				w.WriteHeader(opts.listRemotesStatus)
				_, _ = w.Write([]byte(`{"code":"X","message":"err","details":""}`))
				return
			}
			_, _ = w.Write([]byte(`[` + ditRemoteWithAPI + `]`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/remotes/origin"):
			if opts.getRemoteStatus != 0 {
				w.WriteHeader(opts.getRemoteStatus)
				_, _ = w.Write([]byte(`{"code":"X","message":"err","details":""}`))
				return
			}
			_, _ = w.Write([]byte(ditRemoteWithAPI))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/remotes/origin/commits"):
			if opts.listRemoteCommitsStatus != 0 {
				w.WriteHeader(opts.listRemoteCommitsStatus)
				_, _ = w.Write([]byte(`{"code":"X","message":"err","details":""}`))
				return
			}
			if opts.emptyRemoteCommits {
				_, _ = w.Write([]byte(`[]`))
				return
			}
			_, _ = w.Write([]byte(`[{"id":"c1","properties":{}}]`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pull"):
			if opts.pullStatus != 0 {
				w.WriteHeader(opts.pullStatus)
				_, _ = w.Write([]byte(`{"code":"X","message":"pull-err","details":""}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"op1","type":"PULL","state":"RUNNING","remote":"origin","commitId":"c1"}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/remotes/origin/commits/c1"):
			if opts.getRemoteCommitStatus != 0 {
				w.WriteHeader(opts.getRemoteCommitStatus)
				_, _ = w.Write([]byte(`{"code":"X","message":"err","details":""}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"c1","properties":{}}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/operations/op1/progress"):
			_, _ = w.Write([]byte(`[{"id":1,"type":"COMPLETE"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

type pullOpts struct {
	listRemotesStatus       int
	getRemoteStatus         int
	listRemoteCommitsStatus int
	getRemoteCommitStatus   int
	pullStatus              int
	emptyRemoteCommits      bool
}

func TestPull_ListRemotesError(t *testing.T) {
	port := startMockServer(t, pullServerHandler(t, pullOpts{listRemotesStatus: http.StatusInternalServerError}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Pull("repo", "", "", nil, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on list remotes error; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "remote is not set") {
		t.Errorf("expected error message, got %q", output)
	}
}

func TestPull_GetRemoteError(t *testing.T) {
	port := startMockServer(t, pullServerHandler(t, pullOpts{getRemoteStatus: http.StatusNotFound}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Pull("repo", "", "", nil, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on get remote error; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "remote 'origin' not found") {
		t.Errorf("expected not-found message, got %q", output)
	}
}

func TestPull_TagsWithCommitExits(t *testing.T) {
	port := startMockServer(t, pullServerHandler(t, pullOpts{}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Pull("repo", "guid", "", []string{"v1"}, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on tags+commit; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "tags cannot be specified when commit") {
		t.Errorf("expected tags+commit error, got %q", output)
	}
}

func TestPull_NoMatchingCommits(t *testing.T) {
	port := startMockServer(t, pullServerHandler(t, pullOpts{emptyRemoteCommits: true}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Pull("repo", "", "", nil, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on no matching commits; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "no matching commits found in remote") {
		t.Errorf("expected no-commits message, got %q", output)
	}
}

func TestPull_HappyPath(t *testing.T) {
	port := startMockServer(t, pullServerHandler(t, pullOpts{}))

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() { Pull("repo", "", "", nil, false, port) })
	})

	if didExit {
		t.Errorf("happy path should not exit; output=%q", output)
	}
	if !strings.Contains(output, "Pull completed successfully") {
		t.Errorf("expected success message, got %q", output)
	}
}

func TestPull_WithGuidUsesGetRemoteCommit(t *testing.T) {
	port := startMockServer(t, pullServerHandler(t, pullOpts{}))

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() { Pull("repo", "c1", "", nil, false, port) })
	})

	if didExit {
		t.Errorf("pull with guid happy path should not exit; output=%q", output)
	}
	if !strings.Contains(output, "Pull completed successfully") {
		t.Errorf("expected success message, got %q", output)
	}
}

func TestPull_GetRemoteCommitError(t *testing.T) {
	port := startMockServer(t, pullServerHandler(t, pullOpts{getRemoteCommitStatus: http.StatusInternalServerError}))

	var didExit bool
	var code int
	_ = captureStdout(func() {
		didExit, code = captureExit(t, func() { Pull("repo", "c1", "", nil, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on getRemoteCommit error; got didExit=%v code=%d", didExit, code)
	}
}

func TestPull_PullOperationError(t *testing.T) {
	port := startMockServer(t, pullServerHandler(t, pullOpts{pullStatus: http.StatusInternalServerError}))

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() { Pull("repo", "", "", nil, false, port) })
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on pull error; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "pull-err") {
		t.Errorf("expected server error message, got %q", output)
	}
}

func TestPull_CustomRemoteName(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		remoteJSON := `{"provider":"dit","name":"upstream","properties":{"host":"x","org":"o","repo":"r","api_base_url":"http://api"}}`
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/remotes"):
			_, _ = w.Write([]byte(`[` + remoteJSON + `]`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/remotes/upstream"):
			_, _ = w.Write([]byte(remoteJSON))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/remotes/upstream/commits"):
			_, _ = w.Write([]byte(`[{"id":"c1","properties":{}}]`))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/pull"):
			_, _ = w.Write([]byte(`{"id":"op1","type":"PULL","state":"RUNNING","remote":"upstream","commitId":"c1"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/operations/op1/progress"):
			_, _ = w.Write([]byte(`[{"id":1,"type":"COMPLETE"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	var didExit bool
	_ = captureStdout(func() {
		didExit, _ = captureExit(t, func() { Pull("repo", "", "upstream", nil, false, port) })
	})

	if didExit {
		t.Errorf("custom-remote happy path should not exit")
	}
}
