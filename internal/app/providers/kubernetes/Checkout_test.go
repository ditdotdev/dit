// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"net/http"
	"strings"
	"testing"
)

func TestK8sCheckout_NoCommitsForTags(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, &fakeDocker{}, &fakeK8s{}, func() {
				Checkout("r1", "", []string{"v1"}, port)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "no matching commits found") {
		t.Errorf("expected no-matching message, got %q", output)
	}
}

func TestK8sCheckout_NoSourceCommit(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/status") {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, &fakeDocker{}, &fakeK8s{}, func() {
				Checkout("r1", "", nil, port)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "no commits present") {
		t.Errorf("expected no-commits message, got %q", output)
	}
}

func TestK8sCheckout_TagsAndCommitExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, &fakeDocker{}, &fakeK8s{}, func() {
				Checkout("r1", "abc", []string{"v1"}, port)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "tags and commit cannot both be specified") {
		t.Errorf("expected tags+commit error, got %q", output)
	}
}

func TestK8sCheckout_HappyPath(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/commits/abc/status"):
			_, _ = w.Write([]byte(`{"logicalSize":0,"actualSize":0,"uniqueSize":0,"ready":true}`))
		case strings.Contains(r.URL.Path, "/commits/abc/checkout"):
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/volumes"):
			_, _ = w.Write([]byte(`[{"name":"v0","properties":{"path":"/data"}}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	k := &fakeK8s{}
	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			with(t, &fakeDocker{}, k, func() {
				Checkout("r1", "abc", nil, port)
			})
		})
	})

	if didExit {
		t.Errorf("happy path should not exit; output=%q", output)
	}
	if k.StopPortForwardingCalls != 1 || k.StartPortForwardingCalls != 1 {
		t.Errorf("expected one Stop + one Start port-forward, got %d / %d", k.StopPortForwardingCalls, k.StartPortForwardingCalls)
	}
	if k.UpdateStatefulSetVolumesCall != 1 {
		t.Errorf("expected UpdateStatefulSetVolumes call, got %d", k.UpdateStatefulSetVolumesCall)
	}
	if k.StopStatefulSetCalls != 1 || k.StartStatefulSetCalls != 1 {
		t.Errorf("expected one Stop + one Start statefulset, got %d / %d", k.StopStatefulSetCalls, k.StartStatefulSetCalls)
	}
}

func TestK8sCheckout_CheckoutAPIError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/commits/abc/status"):
			_, _ = w.Write([]byte(`{"logicalSize":0,"actualSize":0,"uniqueSize":0,"ready":true}`))
		case strings.Contains(r.URL.Path, "/commits/abc/checkout"):
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":"X","message":"err","details":""}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, &fakeDocker{}, &fakeK8s{}, func() {
				Checkout("r1", "abc", nil, port)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Error checking out commit abc") {
		t.Errorf("expected checkout error, got %q", output)
	}
}

func TestK8sCheckout_CommitReadyTimeout(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Always not-ready so the loop has to time out.
		if strings.Contains(r.URL.Path, "/commits/abc/status") {
			_, _ = w.Write([]byte(`{"logicalSize":0,"actualSize":0,"uniqueSize":0,"ready":false}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, &fakeDocker{}, &fakeK8s{}, func() {
				Checkout("r1", "abc", nil, port)
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) after timeout; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Timed out") {
		t.Errorf("expected timeout message, got %q", output)
	}
}

func TestK8sCheckout_TagsResolvesToFirstCommit(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/commits") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":"first","properties":{}}]`))
		case strings.HasSuffix(r.URL.Path, "/commits/first/status"):
			_, _ = w.Write([]byte(`{"logicalSize":0,"actualSize":0,"uniqueSize":0,"ready":true}`))
		case strings.Contains(r.URL.Path, "/commits/first/checkout"):
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/volumes"):
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			with(t, &fakeDocker{}, &fakeK8s{}, func() {
				Checkout("r1", "", []string{"v1"}, port)
			})
		})
	})

	if !strings.Contains(output, "Checkout first") {
		t.Errorf("expected checkout-first message, got %q", output)
	}
}
