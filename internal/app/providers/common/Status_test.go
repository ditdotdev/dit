// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestByteCountBinary(t *testing.T) {
	tests := []struct {
		name string
		in   int64
		want string
	}{
		{"zero", 0, "0 B"},
		{"under unit", 512, "512 B"},
		{"just under 1024", 1023, "1023 B"},
		{"exactly 1024", 1024, "1.0 KiB"},
		{"kibibytes", 1500, "1.5 KiB"},
		{"mebibytes", 1024 * 1024 * 2, "2.0 MiB"},
		{"gibibytes", 1024 * 1024 * 1024 * 3, "3.0 GiB"},
		{"tebibytes", int64(1024) * 1024 * 1024 * 1024 * 5, "5.0 TiB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ByteCountBinary(tt.in)
			if got != tt.want {
				t.Errorf("ByteCountBinary(%d) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

type fakeStatusDocker struct {
	stubDocker
	statusByRepo map[string]string
	errByRepo    map[string]error
}

func (f fakeStatusDocker) GetValFromContainer(c string, key ...string) (string, error) {
	if e, ok := f.errByRepo[c]; ok {
		return "", e
	}
	return f.statusByRepo[c], nil
}

func TestGetContainersStatus_HappyPath(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/repositories") {
			_, _ = w.Write([]byte(`[{"name":"r1","properties":{}},{"name":"r2","properties":{}}]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	d := fakeStatusDocker{
		statusByRepo: map[string]string{"r1": "running", "r2": "stopped"},
	}
	var got []runtimeStatus
	withDocker(t, d, func() {
		got = getContainersStatus(port, "ctx")
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0].name != "r1" || got[0].status != "running" {
		t.Errorf("entry 0 = %+v", got[0])
	}
	if got[1].name != "r2" || got[1].status != "stopped" {
		t.Errorf("entry 1 = %+v", got[1])
	}
}

func TestGetContainersStatus_DetachedOnInspectError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/repositories") {
			_, _ = w.Write([]byte(`[{"name":"r1","properties":{}}]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	d := fakeStatusDocker{errByRepo: map[string]error{"r1": errors.New("no such container")}}
	var got []runtimeStatus
	withDocker(t, d, func() {
		got = getContainersStatus(port, "ctx")
	})
	if len(got) != 1 || got[0].status != "detached" {
		t.Errorf("expected detached fallback, got %+v", got)
	}
}

func TestStatus_RepoNotFound(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NotFound","message":"missing","details":""}`))
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, stubDocker{}, func() {
				Status("missing", port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1) on missing repo; got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "repository 'missing' not found") {
		t.Errorf("expected not-found message, got %q", output)
	}
}

func TestStatus_FullOutput(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/repositories/r1/status":
			_, _ = w.Write([]byte(`{"lastCommit":"abc","sourceCommit":"abc"}`))
		case r.URL.Path == "/v1/repositories":
			_, _ = w.Write([]byte(`[{"name":"r1","properties":{}}]`))
		case r.URL.Path == "/v1/repositories/r1/volumes":
			_, _ = w.Write([]byte(`[{"name":"v0","properties":{"path":"/data"}}]`))
		case strings.HasSuffix(r.URL.Path, "/volumes/v0/status"):
			_, _ = w.Write([]byte(`{"name":"v0","logicalSize":2048,"actualSize":1024,"properties":{"path":"/data"},"ready":true}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	d := fakeStatusDocker{statusByRepo: map[string]string{"r1": "running"}}
	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Status("r1", port, "ctx")
			})
		})
	})

	if didExit {
		t.Errorf("Status with healthy repo should not exit")
	}
	for _, want := range []string{"Status:", "running", "Last Commit:", "abc", "Source Commit:", "Volume", "Uncompressed", "Compressed", "/data", "2.0 KiB", "1.0 KiB"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got %q", want, output)
		}
	}
}
