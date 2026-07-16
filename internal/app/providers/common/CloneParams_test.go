// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// Regression for #207: url.Parse errors were swallowed, so a malformed URI
// nil-dereferenced on parsedUri.Path instead of failing cleanly.
func TestClone_InvalidURIExits(t *testing.T) {
	var didExit bool
	var code int
	output := captureStdout(func() {
		withDocker(t, &fakeCloneDocker{}, func() {
			didExit, code = captureExit(t, func() {
				Clone("://bad", "", "", nil, nil, false, nil, 1, "docker", CloneCallbacks{})
			})
		})
	})
	if !didExit || code != 1 {
		t.Errorf("expected exit 1 for invalid URI, got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "invalid URI") {
		t.Errorf("expected 'invalid URI' in output, got %q", output)
	}
}

// Regression for #207: clone's -p provider parameters were silently dropped
// (RemoteAdd received nil). parseRemoteParams mirrors `dit remote add -p`.
func TestParseRemoteParams(t *testing.T) {
	got := parseRemoteParams([]string{"accessKey=abc", "region=us-west-2"})
	want := map[string]string{"accessKey": "abc", "region": "us-west-2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseRemoteParams = %v, want %v", got, want)
	}
	if got := parseRemoteParams(nil); len(got) != 0 {
		t.Errorf("parseRemoteParams(nil) = %v, want empty", got)
	}
}

func TestParseRemoteParams_MalformedExits(t *testing.T) {
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			parseRemoteParams([]string{"noequals"})
		})
	})
	if !didExit || code != 1 {
		t.Errorf("expected exit 1 for malformed parameter, got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("expected key=value format message, got %q", output)
	}
}

// Regression for #207: a failed ListOperations call was swallowed, so abort
// reported as if nothing was running when the server was unreachable.
func TestAbort_ListOperationsErrorExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			Abort("repo", port)
		})
	})
	if !didExit || code != 1 {
		t.Errorf("expected exit 1 when listing operations fails, got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Error listing operations") {
		t.Errorf("expected listing error in output, got %q", output)
	}
}
