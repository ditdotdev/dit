// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

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
		case strings.Contains(r.URL.Path, "/volumes/v0/activate"), strings.Contains(r.URL.Path, "/volumes/v0/deactivate"):
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/volumes/v0") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"name":"v0","properties":{"path":"/data"},"config":{"mountpoint":"/var/lib/dit-docker/mnt/newrepo/v0"}}`))
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
	want := [3]string{"c1", "/data", "/var/lib/dit-docker/mnt/newrepo/v0"}
	if len(d.CpFromContainerCalls) != 1 || d.CpFromContainerCalls[0] != want {
		t.Errorf("expected one CpFromContainer call %v, got %v", want, d.CpFromContainerCalls)
	}
	if d.CpCalls != 0 {
		t.Errorf("host-path Cp must not be used by migrate, got %d calls", d.CpCalls)
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

// migrateMockServer serves the repository create/update and volume
// activate/get/deactivate calls Migrate makes, records every request as
// "METHOD path", and lets a test fail the activate call.
func migrateMockServer(t *testing.T, activateStatus int, requests *[]string) int {
	t.Helper()
	return startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/repositories"):
			_, _ = w.Write([]byte(`{"name":"newrepo","properties":{}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/newrepo"):
			_, _ = w.Write([]byte(`{"name":"newrepo","properties":{}}`))
		case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/repositories/newrepo"):
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/volumes") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"name":"v0","properties":{"path":"/data"}},{"name":"v1","properties":{"path":"/log"}}]`))
		case strings.Contains(r.URL.Path, "/activate"):
			w.WriteHeader(activateStatus)
			if activateStatus != http.StatusOK {
				_, _ = w.Write([]byte(`{"code":"X","message":"zfs mount failed","details":""}`))
			}
		case strings.Contains(r.URL.Path, "/deactivate"):
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/volumes/v0") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"name":"v0","properties":{"path":"/data"},"config":{"mountpoint":"/var/lib/dit-docker/mnt/newrepo/v0"}}`))
		case strings.HasSuffix(r.URL.Path, "/volumes/v1") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"name":"v1","properties":{"path":"/log"},"config":{"mountpoint":"/var/lib/dit-docker/mnt/newrepo/v1"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func twoVolumeFakeDocker() *fakeDocker {
	return &fakeDocker{
		inspectContainerOut: "{}",
		inspectImageOut:     "{}",
		identity:            "docker",
		getValFromContainer: map[string]string{
			"c1:State.Running":           "false",
			"c1:Image":                   "postgres:latest",
			"c1:HostConfig.PortBindings": `{}`,
		},
		getSliceFromImage: map[string][]string{
			"postgres:latest:Config.Volumes": {"/data", "/log"},
			"postgres:latest:RepoDigests":    {"postgres@sha256:abc"},
		},
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// Migrate copies every declared volume path straight out of the source
// container, in declaration order, regardless of how Docker backs it
// (anonymous volume, named volume, bind mount). No host path is involved
// (#256).
func TestMigrate_CopiesFromSourceContainer(t *testing.T) {
	var requests []string
	port := migrateMockServer(t, http.StatusOK, &requests)
	d := twoVolumeFakeDocker()

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	if didExit {
		t.Fatalf("migrate should not exit; output=%q", output)
	}
	want := [][3]string{
		{"c1", "/data", "/var/lib/dit-docker/mnt/newrepo/v0"},
		{"c1", "/log", "/var/lib/dit-docker/mnt/newrepo/v1"},
	}
	if len(d.CpFromContainerCalls) != 2 || d.CpFromContainerCalls[0] != want[0] || d.CpFromContainerCalls[1] != want[1] {
		t.Errorf("expected CpFromContainer calls %v, got %v", want, d.CpFromContainerCalls)
	}
	if d.CpCalls != 0 {
		t.Errorf("host-path Cp must not be used by migrate, got %d calls", d.CpCalls)
	}
	for _, v := range []string{"v0", "v1"} {
		if !contains(requests, "POST /v1/repositories/newrepo/volumes/"+v+"/activate") {
			t.Errorf("expected %s to be activated before the copy; requests=%v", v, requests)
		}
		if !contains(requests, "POST /v1/repositories/newrepo/volumes/"+v+"/deactivate") {
			t.Errorf("expected %s to be deactivated after the copy; requests=%v", v, requests)
		}
	}
	if d.RunCalls != 1 {
		t.Errorf("expected the repository container to be run once, got %d", d.RunCalls)
	}
}

func assertMigrateRolledBack(t *testing.T, d *fakeDocker, requests []string, output string, didExit bool, code int) {
	t.Helper()
	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d output=%q", didExit, code, output)
	}
	if !strings.Contains(output, "Error: failed to copy data to volume v0") {
		t.Errorf("expected copy error, got %q", output)
	}
	if !strings.Contains(output, "Rolled back repository newrepo; c1 is unchanged") {
		t.Errorf("expected rollback message, got %q", output)
	}
	if !contains(requests, "DELETE /v1/repositories/newrepo") {
		t.Errorf("expected repository to be deleted on rollback; requests=%v", requests)
	}
	for _, v := range []string{"v0", "v1"} {
		if !contains(requests, "POST /v1/repositories/newrepo/volumes/"+v+"/deactivate") {
			t.Errorf("expected %s deactivated on rollback; requests=%v", v, requests)
		}
		// Remove passes the server's volume name to docker volume rm, as
		// dit rm does; the e2e rollback case checks the real Docker volume.
		if !contains(d.RemoveVolumeNames, v) {
			t.Errorf("expected volume %s removed on rollback; got %v", v, d.RemoveVolumeNames)
		}
	}
	if d.RunCalls != 0 {
		t.Errorf("repository container must not be started after a failed copy, got %d Run calls", d.RunCalls)
	}
	if strings.Contains(output, "migrated to controlled environment") {
		t.Errorf("must not report success after rollback: %q", output)
	}
}

func TestMigrate_CopyFailureRollsBack(t *testing.T) {
	var requests []string
	port := migrateMockServer(t, http.StatusOK, &requests)
	d := twoVolumeFakeDocker()
	d.cpFromContainerErr = errors.New("Error response from daemon: Could not find the file /data in container c1")

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	assertMigrateRolledBack(t, d, requests, output, didExit, code)
	if !strings.Contains(output, "Could not find the file") {
		t.Errorf("expected the docker error to be surfaced, got %q", output)
	}
}

func TestMigrate_ActivateFailureRollsBack(t *testing.T) {
	var requests []string
	port := migrateMockServer(t, http.StatusInternalServerError, &requests)
	d := twoVolumeFakeDocker()

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	assertMigrateRolledBack(t, d, requests, output, didExit, code)
	if len(d.CpFromContainerCalls) != 0 {
		t.Errorf("no copy should be attempted when activate fails, got %v", d.CpFromContainerCalls)
	}
}
