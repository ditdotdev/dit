// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"net/http"
	"strings"
	"testing"
)

// ---------------------------------------------------------------
// Remove: running/force guard (#207 - the force param was accepted
// and never used, so `dit rm` removed running k8s repos unconditionally)
// ---------------------------------------------------------------

func TestK8sRemove_RunningWithoutForceExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})

	k := &fakeK8s{statefulSetStatus: "running"}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, &fakeDocker{}, k, func() {
				Remove("img", false, port)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected exit 1 removing a running repo without force, got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "is running") {
		t.Errorf("expected 'is running' guidance, got %q", output)
	}
	if k.StopPortForwardingCalls != 0 || k.DeleteStatefulSpecCalls != 0 {
		t.Errorf("guard must fire before any teardown; stopPF=%d deleteSpec=%d",
			k.StopPortForwardingCalls, k.DeleteStatefulSpecCalls)
	}
}

func TestK8sRemove_RunningWithForceProceeds(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/volumes"):
			_, _ = w.Write([]byte(`[]`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	})

	k := &fakeK8s{statefulSetStatus: "running"}
	var didExit bool
	captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			with(t, &fakeDocker{}, k, func() {
				Remove("img", true, port)
			})
		})
	})

	if didExit {
		t.Error("forced remove of a running repo must not exit early")
	}
	if k.StopPortForwardingCalls != 1 || k.DeleteStatefulSpecCalls != 1 {
		t.Errorf("expected teardown to run; stopPF=%d deleteSpec=%d",
			k.StopPortForwardingCalls, k.DeleteStatefulSpecCalls)
	}
}

// ---------------------------------------------------------------
// Run: volume failure paths (#207 - volume-create failure panicked raw
// at the user; volume-status failure orphaned the repo and its volumes)
// ---------------------------------------------------------------

func TestK8sRun_VolumeCreateFailureDeletesRepoAndExits(t *testing.T) {
	repoDeleted := false
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/img/volumes"):
			w.WriteHeader(http.StatusInternalServerError)
		case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/repositories/img"):
			repoDeleted = true
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	})

	d := &fakeDocker{
		inspectImageOut: "{}",
		getSliceFromImage: map[string][]string{
			"img:latest:Config.Volumes": {"/data"},
		},
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Run("img", "", nil, nil, false, false, false, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected clean exit 1 (not a panic) on volume-create failure, got didExit=%v code=%d", didExit, code)
	}
	if !repoDeleted {
		t.Error("repository must be deleted (cascades volume teardown server-side) on volume-create failure")
	}
	if !strings.Contains(output, "Error creating volume") {
		t.Errorf("expected volume error message, got %q", output)
	}
}

func TestK8sRun_VolumeStatusErrorDeletesRepoAndExits(t *testing.T) {
	repoDeleted := false
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/img/volumes"):
			_, _ = w.Write([]byte(`{"name":"v0","properties":{"path":"/data"},"config":{}}`))
		case strings.HasSuffix(r.URL.Path, "/volumes/v0/status"):
			_, _ = w.Write([]byte(`{"name":"v0","logicalSize":0,"actualSize":0,"properties":{},"ready":false,"error":"provisioning failed"}`))
		case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/repositories/img"):
			repoDeleted = true
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	})

	d := &fakeDocker{
		inspectImageOut: "{}",
		getSliceFromImage: map[string][]string{
			"img:latest:Config.Volumes": {"/data"},
		},
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, d, &fakeK8s{}, func() {
				Run("img", "", nil, nil, false, false, false, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected exit 1 on volume provisioning error, got didExit=%v code=%d", didExit, code)
	}
	if !repoDeleted {
		t.Error("repository must be deleted on volume provisioning failure - it used to be orphaned")
	}
	if !strings.Contains(output, "provisioning failed") {
		t.Errorf("expected the provisioning error in output, got %q", output)
	}
}
