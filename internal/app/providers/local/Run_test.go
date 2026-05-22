package local

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestRun_InvalidRepoNameExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				_, _ = Run("img", "bad name!", nil, nil, false, false, false, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if output == "" {
		t.Errorf("expected validation error output, got empty")
	}
}

func TestRun_ContainerExistsExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{containerExists: map[string]bool{"img": true}}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				_, _ = Run("img", "", nil, nil, false, false, false, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "already exists") {
		t.Errorf("expected exists error, got %q", output)
	}
}

func TestRun_ImagePullOnInspectErrorAndEmptyInfoExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{
		inspectImageErr: errors.New("no such image"),
		inspectImageOut: "",
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				_, _ = Run("img", "", nil, nil, false, false, false, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on empty image info; got didExit=%v code=%d", didExit, code)
	}
	if d.PullCalls < 1 {
		t.Errorf("expected Pull attempt after Inspect failure, got %d", d.PullCalls)
	}
	if !strings.Contains(output, "Image information is not available") {
		t.Errorf("expected image-info error, got %q", output)
	}
}

func TestRun_NoVolumesExits(t *testing.T) {
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
			withDocker(t, d, func() {
				_, _ = Run("img", "", nil, nil, false, false, false, port, "ctx")
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

func TestRun_HappyPath(t *testing.T) {
	updatedRepo := false
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/repositories"):
			_, _ = w.Write([]byte(`{"name":"img","properties":{}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/img"):
			updatedRepo = true
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
			"img:latest:RepoDigests": "img@sha256:abc",
		},
	}

	var s string
	var err error
	_ = captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				s, err = Run("img", "", []string{"FOO=bar"}, nil, false, false, true, port, "ctx")
			})
		})
	})

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if !strings.Contains(s, "Running controlled container") {
		t.Errorf("expected running message, got %q", s)
	}
	if !updatedRepo {
		t.Errorf("expected UpdateRepository to be called")
	}
	if d.RunCalls != 1 {
		t.Errorf("expected one Run call, got %d", d.RunCalls)
	}
}

func TestRun_CreateRepoConflictExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/repositories"):
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"code":"Conflict","message":"exists","details":""}`))
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

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				_, _ = Run("img", "", nil, nil, false, false, true, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "already exists") {
		t.Errorf("expected conflict error, got %q", output)
	}
}

func TestRun_CreateVolumeErrorReturnsError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"x","properties":{}}`))
	})

	d := &fakeDocker{
		inspectImageOut: "{}",
		getSliceFromImage: map[string][]string{
			"img:latest:Config.Volumes": {"/data"},
		},
		createVolumeErr: errors.New("volume create failed"),
	}

	var err error
	_ = captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				_, err = Run("img", "", nil, nil, false, false, false, port, "ctx")
			})
		})
	})

	if err == nil {
		t.Errorf("expected error from CreateVolume failure")
	}
}

func TestRun_DockerRunFailure(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"x","properties":{}}`))
	})

	d := &fakeDocker{
		inspectImageOut: "{}",
		getSliceFromImage: map[string][]string{
			"img:latest:Config.Volumes": {"/data"},
		},
		runErr: errors.New("docker run failed"),
	}

	var s string
	var err error
	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				s, err = Run("img", "", nil, nil, false, true, false, port, "ctx")
			})
		})
	})

	if didExit {
		t.Fatalf("unexpected osExit before docker.Run; output=%q", output)
	}
	if err == nil {
		t.Errorf("expected error from Run failure")
	}
	if !strings.Contains(s, "docker run failed") {
		t.Errorf("expected docker run error in output, got %q", s)
	}
}
