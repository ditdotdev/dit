// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

const (
	clonePullImage  = "redis"
	clonePullTag    = "7-alpine"
	clonePullRef    = clonePullImage + ":" + clonePullTag
	clonePullDigest = "redis@sha256:1111"
	clonePullOther  = "redis@sha256:2222"
)

// cloneCommitProps returns v2 commit properties for the clone image tests.
// digest "" omits the key entirely (older servers), which must not load as
// the literal string "<nil>".
func cloneCommitProps(digest string) string {
	digestField := ""
	if digest != "" {
		digestField = `,"digest":"` + digest + `"`
	}
	return `{"v2":{"image":{"image":"` + clonePullImage + `","tag":"` + clonePullTag + `"` + digestField + `},"environment":[],"ports":[],"volumes":[]}}`
}

func runClone(t *testing.T, props string, d *fakeCloneDocker, pull PullPolicy) (didExit bool, code int, output string, calls *cloneCallCounts) {
	t.Helper()
	port := startMockServer(t, statefulCloneWithProps(true, props))
	cb, c := makeCallbacks(nil)
	output = captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Clone("http://example.com/o/myrepo", "", "", nil, nil, false, nil, pull, port, "ctx", cb)
			})
		})
	})
	return didExit, code, output, c
}

func TestClone_PullAlwaysMatchingDigestNoPull(t *testing.T) {
	d := &fakeCloneDocker{repoDigests: `["` + clonePullDigest + `"]`}
	didExit, code, output, calls := runClone(t, cloneCommitProps(clonePullDigest), d, PullAlways)
	if didExit {
		t.Fatalf("unexpected exit %d: %q", code, output)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("matching digest must not pull, got %v", d.pulledRefs)
	}
	if calls.runCalls != 1 {
		t.Errorf("expected one Run call, got %d", calls.runCalls)
	}
}

func TestClone_PullAlwaysMismatchPullsByDigestOnce(t *testing.T) {
	// The inner Run callback must not pull again after Clone resolved the image.
	d := &fakeCloneDocker{repoDigests: `["` + clonePullOther + `"]`}
	didExit, code, output, calls := runClone(t, cloneCommitProps(clonePullDigest), d, PullAlways)
	if didExit {
		t.Fatalf("unexpected exit %d: %q", code, output)
	}
	if len(d.pulledRefs) != 1 || d.pulledRefs[0] != clonePullDigest {
		t.Errorf("expected exactly one pull by digest, got %v", d.pulledRefs)
	}
	if calls.runCalls != 1 {
		t.Errorf("expected one Run call, got %d", calls.runCalls)
	}
}

func TestClone_PullNeverAbsentExits(t *testing.T) {
	d := &fakeCloneDocker{inspectErr: errors.New("no such image")}
	didExit, code, output, calls := runClone(t, cloneCommitProps(clonePullDigest), d, PullNever)
	if !didExit || code != 1 {
		t.Fatalf("expected osExit(1), got didExit=%v code=%d output=%q", didExit, code, output)
	}
	if !strings.Contains(output, "not present locally") {
		t.Errorf("expected --pull never message, got %q", output)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("--pull never must not pull, got %v", d.pulledRefs)
	}
	if calls.runCalls != 0 {
		t.Errorf("Run must not be called, got %d", calls.runCalls)
	}
}

func TestClone_PullNeverMismatchWarnsAndRuns(t *testing.T) {
	d := &fakeCloneDocker{repoDigests: `["` + clonePullOther + `"]`}
	didExit, code, output, calls := runClone(t, cloneCommitProps(clonePullDigest), d, PullNever)
	if didExit {
		t.Fatalf("unexpected exit %d: %q", code, output)
	}
	if !strings.Contains(output, "differs from the digest recorded") {
		t.Errorf("expected mismatch warning, got %q", output)
	}
	if calls.runCalls != 1 {
		t.Errorf("expected one Run call, got %d", calls.runCalls)
	}
}

func TestClone_MissingDigestKeyPullsByTag(t *testing.T) {
	// Older servers may omit the digest; "always" must then pull by tag, not
	// by the string "<nil>".
	d := &fakeCloneDocker{repoDigests: `["` + clonePullOther + `"]`}
	didExit, code, output, _ := runClone(t, cloneCommitProps(""), d, PullAlways)
	if didExit {
		t.Fatalf("unexpected exit %d: %q", code, output)
	}
	if len(d.pulledRefs) != 1 || d.pulledRefs[0] != clonePullRef {
		t.Errorf("expected a single pull by tag %s, got %v", clonePullRef, d.pulledRefs)
	}
}

// Clone must forward the URI's userinfo to RemoteAdd. The ssh provider
// rejects a URL without a username, so stripping "user:pw@" made
// `dit clone ssh://root:root@host/path` fail with "no remote provider found".
func TestClone_RemoteUriKeepsUserinfoDropsQueryAndFragment(t *testing.T) {
	parsed, err := url.Parse("ssh://root:secret@example.com:2222/path/myrepo?tag=v1#c1")
	if err != nil {
		t.Fatal(err)
	}
	got := remoteUriForAdd(parsed)
	want := "ssh://root:secret@example.com:2222/path/myrepo" //nolint:gosec // test fixture, not a credential
	if got != want {
		t.Errorf("remoteUriForAdd() = %q, want %q", got, want)
	}
}
