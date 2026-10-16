// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/ditdotdev/dit/internal/app/providers/common"
)

// The kubernetes Run exits at "No volumes found" once the image is resolved,
// which is enough to prove how the pull policy was applied: the fake records
// every pull, and the no-volumes exit shows the image step was passed.

func k8sRunWithPolicy(t *testing.T, d *fakeDocker, pull common.PullPolicy) (didExit bool, code int, output string) {
	t.Helper()
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	output = captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Run("img", "", nil, nil, false, false, pull, false, port, "ctx")
			})
		})
	})
	return didExit, code, output
}

func TestK8sRun_PullMissingLocalImageNoPull(t *testing.T) {
	d := &fakeDocker{inspectImageOut: "{}", pullErr: errors.New("manifest unknown")}
	didExit, code, output := k8sRunWithPolicy(t, d, common.PullMissing)
	if !didExit || code != 1 || !strings.Contains(output, "No volumes found") {
		t.Fatalf("expected to reach the no-volumes exit, got didExit=%v code=%d output=%q", didExit, code, output)
	}
	if d.PullCalls != 0 {
		t.Errorf("expected no pull for a present image, got %d", d.PullCalls)
	}
}

func TestK8sRun_PullNeverAbsentExits(t *testing.T) {
	d := &fakeDocker{inspectImageErr: errors.New("no such image")}
	didExit, code, output := k8sRunWithPolicy(t, d, common.PullNever)
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

func TestK8sRun_PullAlwaysPullsTag(t *testing.T) {
	d := &fakeDocker{inspectImageOut: "{}"}
	didExit, code, output := k8sRunWithPolicy(t, d, common.PullAlways)
	if !didExit || code != 1 || !strings.Contains(output, "No volumes found") {
		t.Fatalf("expected to reach the no-volumes exit, got didExit=%v code=%d output=%q", didExit, code, output)
	}
	if !reflect.DeepEqual(d.PulledRefs, []string{"img:latest"}) {
		t.Errorf("expected one pull by tag, got %v", d.PulledRefs)
	}
}
