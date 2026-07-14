// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

type fakeCloneDocker struct {
	stubDocker
	inspectErr error
	pullErr    error
	pullCalls  int
}

func (f *fakeCloneDocker) InspectImage(image string) (string, error) { return "", f.inspectErr }
func (f *fakeCloneDocker) Pull(image string) (string, error) {
	f.pullCalls++
	return "", f.pullErr
}

func makeCallbacks(runErr error) (cb CloneCallbacks, calls *cloneCallCounts) {
	c := &cloneCallCounts{}
	cb = CloneCallbacks{
		Run: func(image, repoName string, envs, args []string, disablePortMap, privileged bool) (string, error) {
			c.runCalls++
			return "ran", runErr
		},
		Checkout: func(repoName, commitId string) { c.checkoutCalls++ },
		Remove:   func(repoName string, force bool) { c.removeCalls++ },
	}
	return cb, c
}

type cloneCallCounts struct {
	runCalls      int
	checkoutCalls int
	removeCalls   int
}

func TestClone_CreateRepositoryError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"X","message":"already exists","details":""}`))
	})

	cb, _ := makeCallbacks(nil)
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, &fakeCloneDocker{}, func() {
				Clone("http://x.com/o/myrepo", "", "", nil, nil, false, nil, port, "ctx", cb)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on CreateRepository error; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "error communicating with remote") {
		t.Errorf("expected handleRemoteError output, got %q", output)
	}
}

func TestClone_EmptyRemoteCommitsRemovesRepo(t *testing.T) {
	port := startMockServer(t, statefulClone(false))

	cb, calls := makeCallbacks(nil)
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, &fakeCloneDocker{}, func() {
				Clone("http://example.com/o/myrepo", "", "", nil, nil, false, nil, port, "ctx", cb)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) via removeRepo; got didExit=%v code=%d", didExit, code)
	}
	if calls.removeCalls != 1 {
		t.Errorf("expected Remove callback to be called once, got %d", calls.removeCalls)
	}
	if !strings.Contains(output, "unable to find any matching commits") {
		t.Errorf("expected empty-commits message, got %q", output)
	}
}

func TestClone_ImagePullFailsExits(t *testing.T) {
	port := startMockServer(t, statefulClone(true))

	cb, _ := makeCallbacks(nil)
	d := &fakeCloneDocker{
		inspectErr: errors.New("not local"),
		pullErr:    errors.New("registry down"),
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Clone("http://example.com/o/myrepo", "", "", nil, nil, false, nil, port, "ctx", cb)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on image pull failure; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Unable to find image") {
		t.Errorf("expected pull-failure message, got %q", output)
	}
}

func TestClone_RunCallbackErrorRemovesRepo(t *testing.T) {
	port := startMockServer(t, statefulClone(true))

	cb, calls := makeCallbacks(errors.New("docker run failed"))
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, &fakeCloneDocker{}, func() {
				Clone("http://example.com/o/myrepo", "", "", nil, nil, false, nil, port, "ctx", cb)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) when Run callback fails; got didExit=%v code=%d", didExit, code)
	}
	if calls.runCalls != 1 {
		t.Errorf("expected Run to be called once, got %d", calls.runCalls)
	}
	if calls.removeCalls != 1 {
		t.Errorf("expected Remove to be called once on Run failure, got %d", calls.removeCalls)
	}
	if !strings.Contains(output, "failed to run container") {
		t.Errorf("expected run-failure message, got %q", output)
	}
}

func TestClone_TagsWithCommitIdWarn(t *testing.T) {
	port := startMockServer(t, statefulClone(true))

	cb, _ := makeCallbacks(nil)
	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, &fakeCloneDocker{}, func() {
				Clone("http://example.com/o/myrepo", "", "c1", nil, nil, false, []string{"v1"}, port, "ctx", cb)
			})
		})
	})

	if !strings.Contains(output, "tags cannot be specified with commit ID") {
		t.Errorf("expected tags+commit warning, got %q", output)
	}
}

func TestClone_HappyPath(t *testing.T) {
	port := startMockServer(t, statefulClone(true))

	cb, calls := makeCallbacks(nil)
	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			withDocker(t, &fakeCloneDocker{}, func() {
				Clone("http://example.com/o/myrepo", "", "", nil, nil, false, nil, port, "ctx", cb)
			})
		})
	})

	if didExit {
		t.Errorf("happy path should not exit; output=%q", output)
	}
	if calls.runCalls != 1 {
		t.Errorf("expected one Run call, got %d", calls.runCalls)
	}
	if calls.checkoutCalls != 1 {
		t.Errorf("expected one Checkout call, got %d", calls.checkoutCalls)
	}
}

func TestClone_GetRemoteCommitErrorRemovesRepo(t *testing.T) {
	port := startMockServer(t, cloneHandlerWithCommitError())

	cb, calls := makeCallbacks(nil)
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, &fakeCloneDocker{}, func() {
				Clone("http://example.com/o/myrepo", "", "c1", nil, nil, false, nil, port, "ctx", cb)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if calls.removeCalls != 1 {
		t.Errorf("expected Remove to be called once, got %d", calls.removeCalls)
	}
	if !strings.Contains(output, "error communicating with remote") {
		t.Errorf("expected remote-error message, got %q", output)
	}
}

// cloneRemoteJSON includes the api_base_url so the inner Pull call from Clone
// can hit ensureRemoteRepoExists (which Push uses, not Pull — but the field
// is harmless to include here).
const cloneRemoteJSON = `{"provider":"dit","name":"origin","properties":{"host":"example.com","org":"o","repo":"myrepo"}}`

// cloneHandlerWithCommitError reuses statefulClone(true) but injects an
// error response specifically for the GetRemoteCommit endpoint so callers
// can exercise Clone's handleRemoteError + removeRepo branch.
func cloneHandlerWithCommitError() http.HandlerFunc {
	base := statefulClone(true)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/remotes/origin/commits/c1") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":"X","message":"err","details":""}`))
			return
		}
		base(w, r)
	}
}

// statefulClone returns a handler that simulates RemoteAdd's "doesn't exist
// yet → create → exists" transition: the first GET /remotes/origin returns
// 404 (so RemoteAdd creates the remote), and subsequent GETs return the
// remote. If hasCommit is true the /remotes/origin/commits list returns one
// commit; otherwise an empty list (triggering removeRepo).
func statefulClone(hasCommit bool) http.HandlerFunc {
	createdRemote := false
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/repositories"):
			_, _ = w.Write([]byte(`{"name":"myrepo","properties":{}}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/repositories/myrepo"):
			_, _ = w.Write([]byte(`{"name":"myrepo","properties":{}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/myrepo"):
			_, _ = w.Write([]byte(`{"name":"myrepo","properties":{}}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/remotes"):
			if createdRemote {
				_, _ = w.Write([]byte(`[` + cloneRemoteJSON + `]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/remotes/origin"):
			if createdRemote {
				_, _ = w.Write([]byte(cloneRemoteJSON))
			} else {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"code":"NotFound","message":"x","details":""}`))
			}
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/remotes"):
			createdRemote = true
			_, _ = w.Write([]byte(cloneRemoteJSON))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/remotes/origin/commits/c1"):
			_, _ = w.Write([]byte(`{"id":"c1","properties":{}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/remotes/origin/commits"):
			if hasCommit {
				_, _ = w.Write([]byte(`[{"id":"c1","properties":{}}]`))
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pull"):
			_, _ = w.Write([]byte(`{"id":"op1","type":"PULL","state":"RUNNING","remote":"origin","commitId":"c1"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/operations/op1/progress"):
			_, _ = w.Write([]byte(`[{"id":1,"type":"COMPLETE"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}
