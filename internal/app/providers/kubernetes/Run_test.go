// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestK8sRun_AdditionalArgsRejected(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, &fakeDocker{}, &fakeK8s{}, func() {
				Run("img", "", nil, []string{"extra-arg"}, false, false, false, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "doesn't support additional arguments") {
		t.Errorf("expected args error, got %q", output)
	}
}

func TestK8sRun_SlashInRepoName(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, &fakeDocker{}, &fakeK8s{}, func() {
				Run("img", "bad/name", nil, nil, false, false, false, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "cannot contain a slash") {
		t.Errorf("expected slash error, got %q", output)
	}
}

func TestK8sRun_ImagePullErrorExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{
		inspectImageErr: errors.New("no such image"),
		pullErr:         errors.New("registry down"),
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
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Error pulling image") {
		t.Errorf("expected pull error, got %q", output)
	}
}

func TestK8sRun_NoVolumesExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{
		inspectImageOut:   "{}",
		getSliceFromImage: map[string][]string{},
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
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "No volumes found") {
		t.Errorf("expected no-volumes error, got %q", output)
	}
}

func TestK8sRun_HappyPath(t *testing.T) {
	createdVolume := false
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/img/volumes"):
			createdVolume = true
			_, _ = w.Write([]byte(`{"name":"v0","properties":{"path":"/data"},"config":{"pvc":"img-v0"}}`))
		case strings.HasSuffix(r.URL.Path, "/volumes/v0/status"):
			_, _ = w.Write([]byte(`{"name":"v0","logicalSize":0,"actualSize":0,"properties":{"path":"/data"},"ready":true}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/img"):
			_, _ = w.Write([]byte(`{"name":"img","properties":{}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	d := &fakeDocker{
		inspectImageOut: "{}",
		getSliceFromImage: map[string][]string{
			"img:latest:Config.Volumes":      {"/data"},
			"img:latest:Config.ExposedPorts": {"5432/tcp"},
		},
		getValFromImage: map[string]string{
			"img:latest:RepoDigests": `"img@sha256:abc"`,
		},
	}

	k := &fakeK8s{}
	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			with(t, d, k, func() {
				Run("img", "", []string{"FOO=bar"}, nil, false, false, false, port, "ctx")
			})
		})
	})

	if !createdVolume {
		t.Errorf("expected CreateVolume call; output=%q", output)
	}
	if k.CreateStatefulSetCalls != 1 {
		t.Errorf("expected CreateStatefulSet call, got %d", k.CreateStatefulSetCalls)
	}
	if k.WaitForStatefulSetCalls != 1 {
		t.Errorf("expected WaitForStatefulSet call, got %d", k.WaitForStatefulSetCalls)
	}
	if k.StartPortForwardingCalls != 1 {
		t.Errorf("expected StartPortForwarding call (port mapping enabled), got %d", k.StartPortForwardingCalls)
	}
}

func TestK8sRun_DisablePortMapSkipsForwarding(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/img/volumes"):
			_, _ = w.Write([]byte(`{"name":"v0","properties":{"path":"/data"},"config":{"pvc":"img-v0"}}`))
		case strings.HasSuffix(r.URL.Path, "/volumes/v0/status"):
			_, _ = w.Write([]byte(`{"name":"v0","logicalSize":0,"actualSize":0,"properties":{"path":"/data"},"ready":true}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/img"):
			_, _ = w.Write([]byte(`{"name":"img","properties":{}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	d := &fakeDocker{
		inspectImageOut: "{}",
		getSliceFromImage: map[string][]string{
			"img:latest:Config.Volumes": {"/data"},
		},
	}

	k := &fakeK8s{}
	_ = captureStdout(func() {
		_, _ = captureExit(t, func() {
			with(t, d, k, func() {
				Run("img", "", nil, nil, true, false, false, port, "ctx")
			})
		})
	})

	if k.StartPortForwardingCalls != 0 {
		t.Errorf("expected no StartPortForwarding when --disable-port-map, got %d", k.StartPortForwardingCalls)
	}
}

func TestK8sRun_CreateStatefulSetErrorExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/img/volumes"):
			_, _ = w.Write([]byte(`{"name":"v0","properties":{"path":"/data"},"config":{"pvc":"img-v0"}}`))
		case strings.HasSuffix(r.URL.Path, "/volumes/v0/status"):
			_, _ = w.Write([]byte(`{"name":"v0","logicalSize":0,"actualSize":0,"properties":{"path":"/data"},"ready":true}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/img"):
			_, _ = w.Write([]byte(`{"name":"img","properties":{}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	d := &fakeDocker{
		inspectImageOut: "{}",
		getSliceFromImage: map[string][]string{
			"img:latest:Config.Volumes": {"/data"},
		},
	}
	k := &fakeK8s{createStatefulSetErr: errors.New("orphaned resources")}

	var didExit bool
	var code int
	_ = captureStdout(func() {
		didExit, code = captureExit(t, func() {
			with(t, d, k, func() {
				Run("img", "", nil, nil, true, false, false, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
}
