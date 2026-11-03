// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
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
				Migrate("c1", "newrepo", nil, "user", "user@x", noopCommit, port, "ctx")
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
				Migrate("c1", "newrepo", nil, "user", "user@x", noopCommit, port, "ctx")
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
				Migrate("c1", "bad name!", nil, "user", "user@x", noopCommit, port, "ctx")
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
				Migrate("c1", "newrepo", nil, "user", "user@x", noopCommit, port, "ctx")
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
				Migrate("c1", "newrepo", nil, "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "No volumes found") {
		t.Errorf("expected no-volumes error, got %q", output)
	}
	if !strings.Contains(output, "--volume") {
		t.Errorf("expected the no-volumes error to hint at --volume, got %q", output)
	}
}

// An image that declares no VOLUME can still be migrated when the caller
// names the data paths with --volume; each becomes a dit volume in order (#258).
func TestMigrate_VolumeFlagWithoutDeclaredVolumes(t *testing.T) {
	var requests []string
	port := migrateMockServer(t, http.StatusOK, &requests)
	d := twoVolumeFakeDocker()
	d.getSliceFromImage["postgres:latest:Config.Volumes"] = nil

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", []string{"/app/data"}, "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	if didExit {
		t.Fatalf("migrate should not exit; output=%q", output)
	}
	want := [3]string{"c1", "/app/data", "/var/lib/dit-docker/mnt/newrepo/v0"}
	if len(d.CpFromContainerCalls) != 1 || d.CpFromContainerCalls[0] != want {
		t.Errorf("expected CpFromContainer calls [%v], got %v", want, d.CpFromContainerCalls)
	}
	if !strings.Contains(output, "Creating docker volume newrepo_v0 with path /app/data") {
		t.Errorf("expected v0 to be created for /app/data, got %q", output)
	}
	if strings.Contains(output, "No volumes found") {
		t.Errorf("--volume must bypass the no-volumes guard, got %q", output)
	}
}

// When the image declares volumes and --volume is also given, the flag wins
// and the paths in use are printed.
func TestMigrate_VolumeFlagOverridesDeclaredVolumes(t *testing.T) {
	var requests []string
	port := migrateMockServer(t, http.StatusOK, &requests)
	d := twoVolumeFakeDocker()

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", []string{"/custom/a", "/custom/b"}, "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	if didExit {
		t.Fatalf("migrate should not exit; output=%q", output)
	}
	want := [][3]string{
		{"c1", "/custom/a", "/var/lib/dit-docker/mnt/newrepo/v0"},
		{"c1", "/custom/b", "/var/lib/dit-docker/mnt/newrepo/v1"},
	}
	if len(d.CpFromContainerCalls) != 2 || d.CpFromContainerCalls[0] != want[0] || d.CpFromContainerCalls[1] != want[1] {
		t.Errorf("expected CpFromContainer calls %v, got %v", want, d.CpFromContainerCalls)
	}
	for _, declared := range []string{"/data", "/log"} {
		if strings.Contains(output, "with path "+declared) {
			t.Errorf("declared volume %s must not be used when --volume is given, got %q", declared, output)
		}
	}
	if !strings.Contains(output, "Using volume paths from --volume instead of the image's declared volumes: /custom/a, /custom/b") {
		t.Errorf("expected a note listing the override paths, got %q", output)
	}
}

// Every --volume path must be absolute; a relative one is rejected before
// anything is created on the server.
func TestMigrate_VolumeFlagRejectsRelativePath(t *testing.T) {
	var requests []string
	port := migrateMockServer(t, http.StatusOK, &requests)
	d := twoVolumeFakeDocker()

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Migrate("c1", "newrepo", []string{"/ok", "relative/path"}, "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "relative/path") || !strings.Contains(output, "absolute") {
		t.Errorf("expected an absolute-path error naming the bad path, got %q", output)
	}
	if contains(requests, "POST /v1/repositories") {
		t.Errorf("repository must not be created for an invalid --volume; requests=%v", requests)
	}
	if len(d.CpFromContainerCalls) != 0 {
		t.Errorf("nothing should be copied, got %v", d.CpFromContainerCalls)
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
				Migrate("c1", "newrepo", nil, "user", "user@x", noopCommit, port, "ctx")
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
				Migrate("c1", "newrepo", nil, "user", "user@x", commit, port, "ctx")
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

// A privileged source container (Db2 cannot start without it) must be
// migrated to a privileged repository container, and the flag must reach the
// runtime metadata so a later clone keeps it (#262, run 40944006279).
func TestMigrate_KeepsPrivileged(t *testing.T) {
	for _, tc := range []struct {
		name       string
		privileged string
		want       bool
	}{
		{"privileged source", "true", true},
		{"unprivileged source", "false", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var runtime string
			port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/repositories"):
					_, _ = w.Write([]byte(`{"name":"newrepo","properties":{}}`))
				case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repositories/newrepo"):
					var body struct {
						Properties map[string]interface{} `json:"properties"`
					}
					_ = json.NewDecoder(r.Body).Decode(&body)
					runtime, _ = body.Properties["runtime"].(string)
					_, _ = w.Write([]byte(`{"name":"newrepo","properties":{}}`))
				case strings.Contains(r.URL.Path, "/volumes/v0/activate"), strings.Contains(r.URL.Path, "/volumes/v0/deactivate"):
					w.WriteHeader(http.StatusOK)
				case strings.HasSuffix(r.URL.Path, "/volumes/v0") && r.Method == http.MethodGet:
					_, _ = w.Write([]byte(`{"name":"v0","properties":{"path":"/database"},"config":{"mountpoint":"/var/lib/dit-docker/mnt/newrepo/v0"}}`))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			})

			d := &fakeDocker{
				inspectContainerOut: "{}",
				inspectImageOut:     "{}",
				getValFromContainer: map[string]string{
					"c1:State.Running":           "false",
					"c1:Image":                   "icr.io/db2_community/db2:latest",
					"c1:HostConfig.Privileged":   tc.privileged,
					"c1:HostConfig.PortBindings": `{}`,
				},
				getSliceFromImage: map[string][]string{
					"icr.io/db2_community/db2:latest:Config.Volumes": {"/database"},
					"icr.io/db2_community/db2:latest:RepoDigests":    {"icr.io/db2_community/db2@sha256:abc"},
				},
			}

			captureStdout(func() {
				_, _ = captureExit(t, func() {
					withDocker(t, d, func() {
						Migrate("c1", "newrepo", nil, "user", "user@x", func(string, string, []string, string, string, int) {}, port, "ctx")
					})
				})
			})

			if len(d.RunArgs) != 1 {
				t.Fatalf("expected one Run call, got %d", len(d.RunArgs))
			}
			if got := slices.Contains(d.RunArgs[0], "--privileged"); got != tc.want {
				t.Errorf("Run args %v: --privileged present = %v, want %v", d.RunArgs[0], got, tc.want)
			}
			if got := strings.Contains(runtime, "--privileged"); got != tc.want {
				t.Errorf("runtime metadata %q: --privileged present = %v, want %v", runtime, got, tc.want)
			}
		})
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
				Migrate("c1", "newrepo", nil, "user", "user@x", noopCommit, port, "ctx")
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
				Migrate("c1", "newrepo", nil, "user", "user@x", noopCommit, port, "ctx")
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
				Migrate("c1", "newrepo", nil, "user", "user@x", noopCommit, port, "ctx")
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
				Migrate("c1", "newrepo", nil, "user", "user@x", noopCommit, port, "ctx")
			})
		})
	})

	assertMigrateRolledBack(t, d, requests, output, didExit, code)
	if len(d.CpFromContainerCalls) != 0 {
		t.Errorf("no copy should be attempted when activate fails, got %v", d.CpFromContainerCalls)
	}
}
