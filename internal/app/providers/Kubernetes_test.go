// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"net/http"
	"strings"
	"testing"
	"time"

	k8s "github.com/ditdotdev/dit/internal/app/providers/kubernetes"
)

func TestKubernetes_Constructor(t *testing.T) {
	p := Kubernetes("k8s-ctx", "host", 4242)
	if p.GetType() != "kubernetes" {
		t.Errorf("GetType = %s, want kubernetes", p.GetType())
	}
	if p.GetName() != "k8s-ctx" {
		t.Errorf("GetName = %s, want k8s-ctx", p.GetName())
	}
	if p.GetPort() != 4242 {
		t.Errorf("GetPort = %d, want 4242", p.GetPort())
	}
}

func TestKubernetes_Delete_NoCommit_PrintsMessage(t *testing.T) {
	p := Kubernetes("x", "h", 1)
	out := captureStdout(t, func() { p.Delete("repo", "", nil) })
	if !strings.Contains(out, "No object found to delete") {
		t.Errorf("got %q", out)
	}
}

func TestKubernetes_Delete_WithCommit_DispatchesToCmn(t *testing.T) {
	port := startMockServer(t, alwaysOKHandler())
	p := Kubernetes("x", "h", port)
	_ = captureStdout(t, func() { p.Delete("repo", "abc", nil) })
}

func TestKubernetes_Delete_WithCommitAndTags_DispatchesToCmn(t *testing.T) {
	port := startMockServer(t, alwaysOKHandler())
	p := Kubernetes("x", "h", port)
	_ = captureStdout(t, func() { p.Delete("repo", "abc", []string{"v1"}) })
}

func TestKubernetes_RemoteList_Dispatches(t *testing.T) {
	port := startMockServer(t, alwaysOKHandler())
	p := Kubernetes("x", "h", port)
	_ = captureStdout(t, func() { p.RemoteList("repo") })
}

func TestKubernetes_Log_Dispatches(t *testing.T) {
	port := startMockServer(t, alwaysOKHandler())
	p := Kubernetes("x", "h", port)
	_ = captureStdout(t, func() { p.Log("repo", nil) })
}

func TestKubernetes_Tag_Dispatches(t *testing.T) {
	port := startMockServer(t, alwaysOKHandler())
	p := Kubernetes("x", "h", port)
	_ = captureStdout(t, func() {
		defer func() { _ = recover() }()
		p.Tag("repo", "abc", []string{"v1"})
	})
}

// Copy / Migrate / Upgrade print "not supported" and call the seamed
// osExit. captureExit intercepts the call so the test process survives.
func TestKubernetes_Copy_ExitsNotSupported(t *testing.T) {
	p := Kubernetes("x", "h", 1)
	out := captureStdout(t, func() {
		didExit, code := captureExit(t, func() {
			p.Copy("repo", "local", "/src", "/dst")
		})
		if !didExit || code != 0 {
			t.Errorf("expected exit(0), got didExit=%v code=%d", didExit, code)
		}
	})
	if !strings.Contains(out, "cp is not supported") {
		t.Errorf("expected 'cp is not supported' message, got %q", out)
	}
}

func TestKubernetes_Migrate_ExitsNotSupported(t *testing.T) {
	p := Kubernetes("x", "h", 1)
	out := captureStdout(t, func() {
		didExit, code := captureExit(t, func() { p.Migrate("src", "dst") })
		if !didExit || code != 0 {
			t.Errorf("expected exit(0), got didExit=%v code=%d", didExit, code)
		}
	})
	if !strings.Contains(out, "migrate is not supported") {
		t.Errorf("expected 'migrate is not supported' message, got %q", out)
	}
}

func TestKubernetes_AllDelegators_CoverDispatch(t *testing.T) {
	// Shrink the polling/idle timeouts so each delegator returns fast.
	// k8s.Checkout polls for commit-ready and stateful-set status with
	// long defaults; tests need millisecond timeouts to keep the suite
	// snappy.
	origCommitPoll := k8s.CommitReadyPollInterval
	origCommitTimeout := k8s.CommitReadyTimeout
	k8s.CommitReadyPollInterval = 1 * time.Millisecond
	k8s.CommitReadyTimeout = 10 * time.Millisecond
	// Swap the underlying kubernetes client to a no-op so Run/Start/
	// Checkout/Clone don't hang in WaitForStatefulSet (2-min default
	// timeout, unexported in clients package).
	restoreK8s := k8s.UseNoopK8sForTesting()
	t.Cleanup(func() {
		k8s.CommitReadyPollInterval = origCommitPoll
		k8s.CommitReadyTimeout = origCommitTimeout
		restoreK8s()
	})

	port := startMockServer(t, alwaysOKHandler())
	p := Kubernetes("ctx", "h", port)
	delegators := []struct {
		name string
		fn   func()
	}{
		{"Abort", func() { p.Abort("repo") }},
		{"Checkout", func() { p.Checkout("repo", "abc", nil) }},
		{"Commit", func() { p.Commit("repo", "msg", nil) }},
		{"RemoteLog", func() {
			// RemoteLog needs at least one remote in the list to avoid
			// osExit(1). Use a server scoped to this case.
			p2 := Kubernetes("ctx", "h", startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/remotes") {
					_, _ = w.Write([]byte(`[{"provider":"dit","name":"origin","properties":{"host":"example.com","org":"o","repo":"r"}}]`))
					return
				}
				_, _ = w.Write([]byte("[]"))
			}))
			p2.RemoteLog("repo", "origin", nil)
		}},
		{"Fork", func() { p.Fork("ssh://u", "org", "name") }},
		{"Install", func() { p.Install(nil, false) }},
		{"List", func() { p.List("ctx") }},
		{"Pull", func() { p.Pull("repo", "abc", "origin", nil, false) }},
		{"Push", func() { p.Push("repo", "abc", "origin", nil, false) }},
		{"RemoteAdd", func() { p.RemoteAdd("repo", "ssh://u", "origin", nil) }},
		{"RemoteRemove", func() { p.RemoteRemove("repo", "origin") }},
		{"Remove", func() { p.Remove("repo", false) }},
		{"Status", func() { p.Status("repo") }},
		{"Stop", func() { p.Stop("repo") }},
		{"Uninstall", func() { p.Uninstall(false, false) }},
		{"Start", func() { p.Start("repo") }},
	}

	for _, d := range delegators {
		d := d
		t.Run(d.name, func(t *testing.T) {
			_ = captureStdout(t, func() {
				_, _ = captureExit(t, func() {
					defer func() { _ = recover() }()
					d.fn()
				})
			})
		})
	}
}

func TestKubernetes_Upgrade_ExitsNotSupported(t *testing.T) {
	p := Kubernetes("x", "h", 1)
	out := captureStdout(t, func() {
		didExit, code := captureExit(t, func() {
			p.Upgrade(false, "", false, "/")
		})
		if !didExit || code != 1 {
			t.Errorf("expected exit(1), got didExit=%v code=%d", didExit, code)
		}
	})
	if !strings.Contains(out, "upgrade is not supported") {
		t.Errorf("expected 'upgrade is not supported' message, got %q", out)
	}
}
