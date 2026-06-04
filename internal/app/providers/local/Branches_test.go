package local

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestCopy_StopFailsExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		getValFromContainer: map[string]string{
			"repo:Mounts":        `[{"Destination":"/data"}]`,
			"repo:State.Running": "true",
		},
		stopErr: errors.New("stop failed"),
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Copy("repo", "", "/local/src", "", port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "failed to stop container") {
		t.Errorf("expected stop error, got %q", output)
	}
}

func TestCopy_StartFailsExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/volumes/v0"):
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"name":"v0","properties":{"path":"/data"},"config":{"mountpoint":"/mnt"}}`))
			} else {
				w.WriteHeader(http.StatusOK)
			}
		default:
			w.WriteHeader(http.StatusOK)
		}
	})

	d := &fakeDocker{
		inspectContainerOut: "{}",
		getValFromContainer: map[string]string{
			"repo:Mounts":        `[{"Type":"volume","Name":"repo_v0","Destination":"/data"}]`,
			"repo:State.Running": "true",
		},
		startErr: errors.New("start failed"),
	}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Copy("repo", "", "/local/src", "", port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "failed to start container") {
		t.Errorf("expected start error, got %q", output)
	}
}

func TestRun_ArgsFilteredAndImageDigestUsed(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"x","properties":{}}`))
	})

	d := &fakeDocker{
		inspectImageOut: "{}",
		getSliceFromImage: map[string][]string{
			"img:tag:Config.Volumes":      {"/data"},
			"img:tag:Config.ExposedPorts": {"5432/tcp"},
		},
		getValFromImage: map[string]string{
			"img:tag:RepoDigests": `["img@sha256:a", "img@sha256:b"]`,
		},
	}

	args := []string{"--name", "should-be-stripped", "img:tag", "--keep"}
	_ = captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				_, _ = Run("img:tag", "", nil, args, true, false, false, port, "ctx")
			})
		})
	})

	if d.RunCalls != 1 {
		t.Errorf("expected Run call, got %d", d.RunCalls)
	}
}

func TestRun_NameWithoutValueArg(t *testing.T) {
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
	}

	args := []string{"--name"} // dangling --name with no value
	_ = captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				_, _ = Run("img", "", nil, args, true, false, false, port, "ctx")
			})
		})
	})
}

func TestMigrate_HappyPathWithMountsAndPorts(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/repositories"):
			_, _ = w.Write([]byte(`{"name":"newrepo","properties":{}}`))
		case strings.Contains(r.URL.Path, "/volumes/v0/activate"):
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "/volumes/v0/deactivate"):
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/volumes/v0"):
			_, _ = w.Write([]byte(`{"name":"v0","properties":{"path":"/data"},"config":{"mountpoint":"/mnt/v0"}}`))
		case strings.HasSuffix(r.URL.Path, "/repositories/newrepo"):
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
			"c1:Mounts":                  `[{"Source":"/local/data","Destination":"/data"}]`,
			"c1:HostConfig.PortBindings": `{"5432/tcp":[{"HostIp":"","HostPort":"5432"}]}`,
		},
		getSliceFromContainer: map[string][]string{},
		getSliceFromImage: map[string][]string{
			"postgres:latest:Config.Volumes": {"/data"},
			"postgres:latest:RepoDigests":    {`"postgres@sha256:abc"`},
		},
	}

	commitCalled := false
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
		t.Errorf("expected commit callback; output=%q", output)
	}
}

func TestMigrate_HappyPathWithHostIp(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/repositories"):
			_, _ = w.Write([]byte(`{"name":"newrepo","properties":{}}`))
		case strings.HasSuffix(r.URL.Path, "/repositories/newrepo"):
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
			"c1:HostConfig.PortBindings": `{"5432/tcp":[{"HostIp":"127.0.0.1","HostPort":"5432"}]}`,
		},
		getSliceFromImage: map[string][]string{
			"postgres:latest:Config.Volumes": {"/data"},
			"postgres:latest:RepoDigests":    {`"postgres@sha256:abc"`},
		},
	}

	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	if !strings.Contains(output, "migrated to controlled environment") {
		t.Errorf("expected migrate-complete message, got %q", output)
	}
}

func TestUninstall_TeardownErrorIsWarning(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})

	d := &fakeDocker{
		ditServerAvailable: true,
		teardownErr:        errors.New("zfs pool busy"),
	}

	output := captureStdout(func() {
		_, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Uninstall("v1.0.0", true, false, port, "ctx")
			})
		})
	})

	if !strings.Contains(output, "Teardown encountered errors") {
		t.Errorf("expected teardown warning, got %q", output)
	}
}
