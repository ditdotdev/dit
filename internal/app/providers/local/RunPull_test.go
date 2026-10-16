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

func runWithPolicy(t *testing.T, d *fakeDocker, pull common.PullPolicy) (s string, err error, didExit bool, code int, output string) {
	t.Helper()
	port := runPullServer(t)
	output = captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				s, err = Run("img", "", nil, nil, false, false, pull, true, port, "ctx")
			})
		})
	})
	return s, err, didExit, code, output
}

// Issue #247 counterpart for run: a present image with an unreachable
// registry must run without any pull.
func TestRun_PullMissingUsesLocalImageWhenRegistryUnreachable(t *testing.T) {
	d := presentFake()
	d.pullErr = errors.New("manifest unknown")
	s, err, didExit, code, output := runWithPolicy(t, d, common.PullMissing)
	if didExit || err != nil {
		t.Fatalf("unexpected failure: exit=%v code=%d err=%v output=%q", didExit, code, err, output)
	}
	if d.PullCalls != 0 {
		t.Errorf("expected no pull, got %d (%v)", d.PullCalls, d.PulledRefs)
	}
	if !strings.Contains(s, "Running controlled container") {
		t.Errorf("expected running message, got %q", s)
	}
}

func TestRun_PullMissingAbsentPullFailsExits(t *testing.T) {
	d := presentFake()
	d.inspectImageErr = errors.New("no such image")
	d.inspectImageOut = ""
	d.pullErr = errors.New("manifest unknown")
	_, _, didExit, code, output := runWithPolicy(t, d, common.PullMissing)
	if !didExit || code != 1 {
		t.Fatalf("expected osExit(1), got didExit=%v code=%d output=%q", didExit, code, output)
	}
	if !strings.Contains(output, "not found locally") {
		t.Errorf("expected not-found message, got %q", output)
	}
	if d.PullCalls != 1 {
		t.Errorf("expected exactly one pull attempt, got %d", d.PullCalls)
	}
}

func TestRun_PullAlwaysPullsTag(t *testing.T) {
	d := presentFake()
	_, err, didExit, code, output := runWithPolicy(t, d, common.PullAlways)
	if didExit || err != nil {
		t.Fatalf("unexpected failure: exit=%v code=%d err=%v output=%q", didExit, code, err, output)
	}
	if !reflect.DeepEqual(d.PulledRefs, []string{"img:latest"}) {
		t.Errorf("expected one pull by tag, got %v", d.PulledRefs)
	}
}

func TestRun_PullAlwaysPullFailsWarnsAndContinues(t *testing.T) {
	d := presentFake()
	d.pullErr = errors.New("manifest unknown")
	s, err, didExit, code, output := runWithPolicy(t, d, common.PullAlways)
	if didExit || err != nil {
		t.Fatalf("unexpected failure: exit=%v code=%d err=%v output=%q", didExit, code, err, output)
	}
	if !strings.Contains(output, "Warning: unable to pull") {
		t.Errorf("expected pull warning, got %q", output)
	}
	if !strings.Contains(s, "Running controlled container") {
		t.Errorf("expected running message, got %q", s)
	}
}

func TestRun_PullNeverPresentRuns(t *testing.T) {
	d := presentFake()
	_, err, didExit, code, output := runWithPolicy(t, d, common.PullNever)
	if didExit || err != nil {
		t.Fatalf("unexpected failure: exit=%v code=%d err=%v output=%q", didExit, code, err, output)
	}
	if d.PullCalls != 0 {
		t.Errorf("expected no pull, got %d", d.PullCalls)
	}
}

func TestRun_PullNeverAbsentExits(t *testing.T) {
	d := presentFake()
	d.inspectImageErr = errors.New("no such image")
	d.inspectImageOut = ""
	_, _, didExit, code, output := runWithPolicy(t, d, common.PullNever)
	if !didExit || code != 1 {
		t.Fatalf("expected osExit(1), got didExit=%v code=%d output=%q", didExit, code, output)
	}
	if !strings.Contains(output, "not present locally") {
		t.Errorf("expected --pull never message, got %q", output)
	}
	if d.PullCalls != 0 {
		t.Errorf("--pull never must not pull, got %d", d.PullCalls)
	}
}
