// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestGetLocalSrcFromPath(t *testing.T) {
	mounts := []mount{
		{Source: "/a", Destination: "/data"},
		{Source: "/b", Destination: "/log"},
	}
	if got := getLocalSrcFromPath("/data", mounts); got != "/a" {
		t.Errorf("expected /a, got %q", got)
	}
	if got := getLocalSrcFromPath("/missing", mounts); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func noopCommit(string, string, []string, string, string, int) {}

func TestMigrate_InspectContainerError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{inspectContainerErr: errors.New("no such container")}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Container information is not available") {
		t.Errorf("expected container info error, got %q", output)
	}
}

func TestMigrate_RunningContainerExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		getValFromContainer: map[string]string{"c1:State.Running": "true"},
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Cannot migrate a running container") {
		t.Errorf("expected running-container error, got %q", output)
	}
}

func TestMigrate_InvalidRepoName(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		getValFromContainer: map[string]string{"c1:State.Running": "false"},
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "bad name!", "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if output == "" {
		t.Errorf("expected validation error output")
	}
}

func TestMigrate_ImageInspectError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		getValFromContainer: map[string]string{
			"c1:State.Running": "false",
			"c1:Image":         "postgres:latest",
		},
		inspectImageErr: errors.New("no such image"),
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Image information is not available") {
		t.Errorf("expected image-info error, got %q", output)
	}
}

func TestMigrate_NoVolumes(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		inspectImageOut:     "{}",
		getValFromContainer: map[string]string{
			"c1:State.Running": "false",
			"c1:Image":         "postgres:latest",
		},
		getSliceFromImage: map[string][]string{},
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", "user", "user@x", noopCommit, port, "ctx")
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

func TestMigrate_CreateRepoErrorReturns(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/repositories") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":"X","message":"err","details":""}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		inspectImageOut:     "{}",
		getValFromContainer: map[string]string{
			"c1:State.Running": "false",
			"c1:Image":         "postgres:latest",
		},
		getSliceFromImage: map[string][]string{
			"postgres:latest:Config.Volumes": {"/data"},
		},
	}

	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	if !strings.Contains(output, "Error creating repository") {
		t.Errorf("expected create-repo error, got %q", output)
	}
}

func TestMigrate_HappyPath(t *testing.T) {
	commitCalled := false
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/repositories"):
			_, _ = w.Write([]byte(`{"name":"newrepo","properties":{}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/newrepo"):
			_, _ = w.Write([]byte(`{"name":"newrepo","properties":{}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		inspectImageOut:     "{}",
		getValFromContainer: map[string]string{
			"c1:State.Running":           "false",
			"c1:Image":                   "postgres:latest",
			"c1:Mounts":                  `[]`,
			"c1:HostConfig.PortBindings": `{}`,
		},
		getSliceFromContainer: map[string][]string{
			"c1": {`"FOO=bar"`},
		},
		getSliceFromImage: map[string][]string{
			"postgres:latest:Config.Volumes": {"/data"},
			"postgres:latest:RepoDigests":    {"postgres@sha256:abc"},
		},
	}

	commit := func(name, msg string, tags []string, user, email string, port int) {
		commitCalled = true
	}

	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", "user", "user@x", commit, port, "ctx")
			})
		})
	})

	if !commitCalled {
		t.Errorf("expected commit callback to be invoked; output=%q", output)
	}
	if !strings.Contains(output, "migrated to controlled environment newrepo") {
		t.Errorf("expected migrate-complete message, got %q", output)
	}
}

func TestMigrate_RunFailureExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"newrepo","properties":{}}`))
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		inspectImageOut:     "{}",
		getValFromContainer: map[string]string{
			"c1:State.Running":           "false",
			"c1:Image":                   "postgres:latest",
			"c1:Mounts":                  `[]`,
			"c1:HostConfig.PortBindings": `{}`,
		},
		getSliceFromImage: map[string][]string{
			"postgres:latest:Config.Volumes": {"/data"},
			"postgres:latest:RepoDigests":    {"postgres@sha256:abc"},
		},
		runErr: errors.New("docker run failed"),
	}

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on Run failure; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "docker run failed") {
		t.Errorf("expected docker run error, got %q", output)
	}
}
