// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/ditdotdev/dit/internal/app/utils"
)

// init shrinks the OperationMonitor poll interval to a fast value for tests so
// Pull/Push tests don't sleep for seconds between progress polls.
func init() {
	utils.MonitorPollInterval = 1 * time.Millisecond
}

// startMockServer spins up an httptest server with the given handler and
// returns the port number to pass to the provider function under test. The
// server is registered for cleanup via t.Cleanup.
func startMockServer(t *testing.T, handler http.HandlerFunc) int {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse httptest URL %q: %v", srv.URL, err)
	}
	p, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("parse httptest port %q: %v", u.Port(), err)
	}
	return p
}

// exitPanic is the sentinel a swapped osExit panics with so the caller can
// distinguish a captured exit from an unrelated panic.
type exitPanic struct{ code int }

// captureExit runs fn with osExit swapped to a recorder. It returns whether
// fn requested an exit and, if so, what code. Other panics propagate.
func captureExit(t *testing.T, fn func()) (didExit bool, code int) {
	t.Helper()
	originalExit := osExit
	defer func() { osExit = originalExit }()
	osExit = func(c int) { panic(exitPanic{c}) }
	defer func() {
		if r := recover(); r != nil {
			if p, ok := r.(exitPanic); ok {
				didExit = true
				code = p.code
				return
			}
			panic(r)
		}
	}()
	fn()
	return false, 0
}

// stubDocker is the default zero-value implementation of dockerClient used by
// tests that don't care about docker interactions. Embed it and override only
// the methods the test needs.
type stubDocker struct{}

func (stubDocker) GetValFromContainer(c string, key ...string) (string, error) {
	return "", nil
}
func (stubDocker) InspectImage(image string) (string, error) { return "", nil }
func (stubDocker) Pull(image string) (string, error)         { return "", nil }

// withDocker swaps newDocker for the duration of fn. Returns no value; just
// for ergonomics inside test bodies.
func withDocker(t *testing.T, d dockerClient, fn func()) {
	t.Helper()
	originalNewDocker := newDocker
	defer func() { newDocker = originalNewDocker }()
	newDocker = func(string, int) dockerClient { return d }
	fn()
}
