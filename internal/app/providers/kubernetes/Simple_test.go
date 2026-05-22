package kubernetes

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestList_PrintsReposWithStatus(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"r1","properties":{}},{"name":"r2","properties":{}}]`))
	})

	k := &fakeK8s{statefulSetStatus: "running"}
	output := captureStdout(func() {
		with(t, &fakeDocker{}, k, func() {
			List("k8s-ctx", port)
		})
	})

	if !strings.Contains(output, "r1") || !strings.Contains(output, "running") {
		t.Errorf("expected r1 + running, got %q", output)
	}
}

func TestList_DetachedFallbackOnError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"r1","properties":{}}]`))
	})

	k := &fakeK8s{statefulSetStatusErr: errors.New("no such statefulset")}
	output := captureStdout(func() {
		with(t, &fakeDocker{}, k, func() {
			List("k8s-ctx", port)
		})
	})

	if !strings.Contains(output, "detached") {
		t.Errorf("expected detached fallback, got %q", output)
	}
}

func TestStart_StartsStatefulSetAndPortForward(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"r1","properties":{}}`))
	})

	k := &fakeK8s{}
	output := captureStdout(func() {
		with(t, &fakeDocker{}, k, func() {
			Start("r1", port)
		})
	})

	if k.StartStatefulSetCalls != 1 {
		t.Errorf("expected StartStatefulSet call, got %d", k.StartStatefulSetCalls)
	}
	if k.StartPortForwardingCalls != 1 {
		t.Errorf("expected StartPortForwarding call, got %d", k.StartPortForwardingCalls)
	}
	if !strings.Contains(output, "Updating deployment") {
		t.Errorf("expected start message, got %q", output)
	}
}

func TestStart_SkipsPortForwardWhenDisabled(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"r1","properties":{"v2":{"disablePortMapping":true,"image":{"image":"i","tag":"t","digest":""},"environment":[],"ports":[],"volumes":[]}}}`))
	})

	k := &fakeK8s{}
	_ = captureStdout(func() {
		with(t, &fakeDocker{}, k, func() {
			Start("r1", port)
		})
	})

	if k.StartPortForwardingCalls != 0 {
		t.Errorf("expected no StartPortForwarding when disabled, got %d", k.StartPortForwardingCalls)
	}
}

func TestStop_StopsStatefulSetAndPortForward(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"r1","properties":{}}`))
	})

	k := &fakeK8s{}
	output := captureStdout(func() {
		with(t, &fakeDocker{}, k, func() {
			Stop("r1", port)
		})
	})

	if k.StopStatefulSetCalls != 1 {
		t.Errorf("expected StopStatefulSet call, got %d", k.StopStatefulSetCalls)
	}
	if k.StopPortForwardingCalls != 1 {
		t.Errorf("expected StopPortForwarding call, got %d", k.StopPortForwardingCalls)
	}
	if !strings.Contains(output, "Stopped r1") {
		t.Errorf("expected stop message, got %q", output)
	}
}

func TestRemove_HappyPath(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/volumes") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"name":"v0","properties":{"path":"/data"}}]`))
		case strings.Contains(r.URL.Path, "/volumes/v0") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/repositories/r1") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	k := &fakeK8s{}
	output := captureStdout(func() {
		with(t, &fakeDocker{}, k, func() {
			Remove("r1", false, port)
		})
	})

	if k.DeleteStatefulSpecCalls != 1 {
		t.Errorf("expected DeleteStatefulSpec call, got %d", k.DeleteStatefulSpecCalls)
	}
	if !strings.Contains(output, "r1 removed") {
		t.Errorf("expected remove message, got %q", output)
	}
}

func TestRemove_DeleteRepoFailureReturns(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/volumes"):
			_, _ = w.Write([]byte(`[]`))
		case strings.HasSuffix(r.URL.Path, "/repositories/r1") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NotFound","message":"missing","details":""}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	output := captureStdout(func() {
		with(t, &fakeDocker{}, &fakeK8s{}, func() {
			Remove("r1", false, port)
		})
	})

	if !strings.Contains(output, "Error deleting repository r1") {
		t.Errorf("expected delete error, got %q", output)
	}
}

func TestRemove_VolumeDeleteWarning(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/volumes") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"name":"v0","properties":{"path":"/data"}}]`))
		case strings.Contains(r.URL.Path, "/volumes/v0") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":"X","message":"err","details":""}`))
		case strings.HasSuffix(r.URL.Path, "/repositories/r1") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	output := captureStdout(func() {
		with(t, &fakeDocker{}, &fakeK8s{}, func() {
			Remove("r1", false, port)
		})
	})

	if !strings.Contains(output, "Warning: Failed to delete volume v0") {
		t.Errorf("expected volume delete warning, got %q", output)
	}
}
