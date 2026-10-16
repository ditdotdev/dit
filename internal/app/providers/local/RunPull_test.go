// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/ditdotdev/dit/internal/app/providers/common"
)

// runPullServer accepts repository create/update so Run reaches docker run.
func runPullServer(t *testing.T) int {
	t.Helper()
	return startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/repositories"):
			_, _ = w.Write([]byte(`{"name":"img","properties":{}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/img"):
			_, _ = w.Write([]byte(`{"name":"img","properties":{}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func presentFake() *fakeDocker {
	return &fakeDocker{
		inspectImageOut: "{}",
		getSliceFromImage: map[string][]string{
			"img:latest:Config.Volumes":      {"/data"},
			"img:latest:Config.ExposedPorts": {"5432/tcp"},
		},
		getValFromImage: map[string]string{"img:latest:RepoDigests": "[]"},
	}
}

type runResult struct {
	s       string
	err     error
	didExit bool
	code    int
	output  string
}

func runWithPolicy(t *testing.T, d *fakeDocker, pull common.PullPolicy) runResult {
	var r runResult
	t.Helper()
	port := runPullServer(t)
	r.output = captureStdout(func() {
		r.didExit, r.code = captureExit(t, func() {
			withDocker(t, d, func() {
				r.s, r.err = Run("img", "", nil, nil, false, false, pull, true, port, "ctx")
			})
		})
	})
	return r
}

// Issue #247 counterpart for run: a present image with an unreachable
// registry must run without any pull.
func TestRun_PullMissingUsesLocalImageWhenRegistryUnreachable(t *testing.T) {
	d := presentFake()
	d.pullErr = errors.New("manifest unknown")
	r := runWithPolicy(t, d, common.PullMissing)
	if r.didExit || r.err != nil {
		t.Fatalf("unexpected failure: exit=%v code=%d err=%v output=%q", r.didExit, r.code, r.err, r.output)
	}
	if d.PullCalls != 0 {
		t.Errorf("expected no pull, got %d (%v)", d.PullCalls, d.PulledRefs)
	}
	if !strings.Contains(r.s, "Running controlled container") {
		t.Errorf("expected running message, got %q", r.s)
	}
}

func TestRun_PullMissingAbsentPullFailsExits(t *testing.T) {
	d := presentFake()
	d.inspectImageErr = errors.New("no such image")
	d.inspectImageOut = ""
	d.pullErr = errors.New("manifest unknown")
	r := runWithPolicy(t, d, common.PullMissing)
	if !r.didExit || r.code != 1 {
		t.Fatalf("expected osExit(1), got didExit=%v code=%d output=%q", r.didExit, r.code, r.output)
	}
	if !strings.Contains(r.output, "not found locally") {
		t.Errorf("expected not-found message, got %q", r.output)
	}
	if d.PullCalls != 1 {
		t.Errorf("expected exactly one pull attempt, got %d", d.PullCalls)
	}
}

// Issue #240: clone passes image:tag@digest. Run must inspect that
// reference (not image:tag, which may point at a different image), record
// the committed digest, and start the container from it without pulling.
func TestRun_DigestReferenceRunsCommittedImage(t *testing.T) {
	const ref = "img:1.0@sha256:abc"
	d := &fakeDocker{
		inspectImageOut: "{}",
		getSliceFromImage: map[string][]string{
			ref + ":Config.Volumes":      {"/data"},
			ref + ":Config.ExposedPorts": {"5432/tcp"},
		},
		getValFromImage: map[string]string{ref + ":RepoDigests": `["img@sha256:abc"]`},
		pullErr:         errors.New("must not pull"),
	}
	port := runPullServer(t)
	var r runResult
	r.output = captureStdout(func() {
		r.didExit, r.code = captureExit(t, func() {
			withDocker(t, d, func() {
				r.s, r.err = Run(ref, "img", nil, []string{ref, "-e", "X=1"}, false, false, common.PullMissing, true, port, "ctx")
			})
		})
	})
	if r.didExit || r.err != nil {
		t.Fatalf("unexpected failure: exit=%v code=%d err=%v output=%q", r.didExit, r.code, r.err, r.output)
	}
	if d.PullCalls != 0 {
		t.Errorf("expected no pull for a present digest, got %d (%v)", d.PullCalls, d.PulledRefs)
	}
	if !reflect.DeepEqual(d.RunImages, []string{"img@sha256:abc"}) {
		t.Errorf("expected docker run of the committed digest, got %v", d.RunImages)
	}
}

func TestRun_PullAlwaysPullsTag(t *testing.T) {
	d := presentFake()
	r := runWithPolicy(t, d, common.PullAlways)
	if r.didExit || r.err != nil {
		t.Fatalf("unexpected failure: exit=%v code=%d err=%v output=%q", r.didExit, r.code, r.err, r.output)
	}
	if !reflect.DeepEqual(d.PulledRefs, []string{"img:latest"}) {
		t.Errorf("expected one pull by tag, got %v", d.PulledRefs)
	}
}

func TestRun_PullAlwaysPullFailsWarnsAndContinues(t *testing.T) {
	d := presentFake()
	d.pullErr = errors.New("manifest unknown")
	r := runWithPolicy(t, d, common.PullAlways)
	if r.didExit || r.err != nil {
		t.Fatalf("unexpected failure: exit=%v code=%d err=%v output=%q", r.didExit, r.code, r.err, r.output)
	}
	if !strings.Contains(r.output, "Warning: unable to pull") {
		t.Errorf("expected pull warning, got %q", r.output)
	}
	if !strings.Contains(r.s, "Running controlled container") {
		t.Errorf("expected running message, got %q", r.s)
	}
}

func TestRun_PullNeverPresentRuns(t *testing.T) {
	d := presentFake()
	r := runWithPolicy(t, d, common.PullNever)
	if r.didExit || r.err != nil {
		t.Fatalf("unexpected failure: exit=%v code=%d err=%v output=%q", r.didExit, r.code, r.err, r.output)
	}
	if d.PullCalls != 0 {
		t.Errorf("expected no pull, got %d", d.PullCalls)
	}
}

func TestRun_PullNeverAbsentExits(t *testing.T) {
	d := presentFake()
	d.inspectImageErr = errors.New("no such image")
	d.inspectImageOut = ""
	r := runWithPolicy(t, d, common.PullNever)
	if !r.didExit || r.code != 1 {
		t.Fatalf("expected osExit(1), got didExit=%v code=%d output=%q", r.didExit, r.code, r.output)
	}
	if !strings.Contains(r.output, "not present locally") {
		t.Errorf("expected --pull never message, got %q", r.output)
	}
	if d.PullCalls != 0 {
		t.Errorf("--pull never must not pull, got %d", d.PullCalls)
	}
}
