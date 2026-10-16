// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package clients

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// Tests for #244: `dit uninstall` hung for 17 hours because the ZFS
// teardown container was run with `--rm` and the docker daemon never
// finished the auto-remove, so the waiting `docker run` client never
// returned. TeardownDitServers must therefore:
//
//   - bound the wait on `docker run` (teardownTimeout)
//   - run the container under a known name instead of relying on `--rm`
//   - force-remove that container itself, before and after the run
//   - surface the output the teardown script already produced on timeout

const teardownContainer = "dit-test-id-teardown"

// withTeardownTimeout overrides the package-level teardown deadline for
// the duration of the test.
func withTeardownTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	orig := teardownTimeout
	teardownTimeout = d
	t.Cleanup(func() { teardownTimeout = orig })
}

// callTeardown runs TeardownDitServers in a goroutine and fails the test
// if it does not return within the given guard. The guard is the
// replication of the bug: before the fix the call blocks forever on the
// hung `docker run`.
func callTeardown(t *testing.T, d docker, guard time.Duration) (string, error) {
	t.Helper()
	type result struct {
		out string
		err error
	}
	done := make(chan result, 1)
	go func() {
		out, err := d.TeardownDitServers()
		done <- result{out, err}
	}()
	select {
	case r := <-done:
		return r.out, r.err
	case <-time.After(guard):
		t.Fatalf("TeardownDitServers did not return within %s: uninstall would hang (#244)", guard)
		return "", nil
	}
}

func hangingFake(t *testing.T) *fakeCE {
	t.Helper()
	fake := &fakeCE{
		hangOnRun:  make(chan struct{}),
		partialOut: "Removing docker plugin socket\n",
	}
	// Release the blocked goroutine when the test ends so it does not leak
	// into later tests.
	t.Cleanup(func() { close(fake.hangOnRun) })
	withFakeCE(t, fake)
	return fake
}

func findCall(calls [][]string, want ...string) int {
	for i, c := range calls {
		if strings.Join(c, " ") == strings.Join(want, " ") {
			return i
		}
	}
	return -1
}

func findRun(calls [][]string) int {
	for i, c := range calls {
		if len(c) > 1 && c[0] == dockerCmd && c[1] == dockerRun {
			return i
		}
	}
	return -1
}

func TestDocker_TeardownDitServers_ReturnsWhenDockerRunHangs(t *testing.T) {
	fake := hangingFake(t)
	withTeardownTimeout(t, 50*time.Millisecond)
	d := Docker("test-id", 5001)

	_, err := callTeardown(t, d, 2*time.Second)

	if err == nil {
		t.Fatal("expected an error when the teardown container never exits")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error should wrap context.DeadlineExceeded, got %v", err)
	}
	if !strings.Contains(err.Error(), teardownContainer) {
		t.Errorf("error should name the stuck container %q so the user can address it, got %v", teardownContainer, err)
	}

	run := findRun(fake.calls)
	rm := findCall(fake.calls, dockerCmd, "rm", "-f", teardownContainer)
	if run < 0 {
		t.Fatalf("expected a docker run call, got %v", fake.calls)
	}
	// The force-remove after the run is what unwedged the daemon in the
	// incident; it must happen even when the run timed out.
	if rm < 0 {
		t.Fatalf("expected `docker rm -f %s` after timeout, got %v", teardownContainer, fake.calls)
	}
	if last := findLastCall(fake.calls, dockerCmd, "rm", "-f", teardownContainer); last < run {
		t.Errorf("force-remove must follow the hung run: run=%d rm=%d calls=%v", run, last, fake.calls)
	}
}

func findLastCall(calls [][]string, want ...string) int {
	last := -1
	for i, c := range calls {
		if strings.Join(c, " ") == strings.Join(want, " ") {
			last = i
		}
	}
	return last
}

func TestDocker_TeardownDitServers_TimeoutErrorIncludesPartialOutput(t *testing.T) {
	hangingFake(t)
	withTeardownTimeout(t, 50*time.Millisecond)
	d := Docker("test-id", 5001)

	out, err := callTeardown(t, d, 2*time.Second)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	// In the incident the script had already printed its final line; that
	// output is the only clue the user gets, so it must be returned.
	if !strings.Contains(out, "Removing docker plugin socket") {
		t.Errorf("expected the teardown script's partial output to be returned, got %q", out)
	}
}

func TestDocker_TeardownDitServers_UsesNamedContainerWithoutRm(t *testing.T) {
	fake := &fakeCE{out: "ok"}
	withFakeCE(t, fake)
	d := Docker("test-id", 5001)

	if _, err := d.TeardownDitServers(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	run := findRun(fake.calls)
	if run < 0 {
		t.Fatalf("expected a docker run call, got %v", fake.calls)
	}
	args := fake.calls[run]
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "--name="+teardownContainer) {
		t.Errorf("teardown container must be addressable by name %q, args: %v", teardownContainer, args)
	}
	for _, forbidden := range []string{"--rm", "-d", "--restart", "always", "--name=dit-test-id-launch"} {
		for _, a := range args {
			if a == forbidden {
				t.Errorf("teardown run args must not contain %q (the daemon's --rm auto-remove is the step that wedged): %v", forbidden, args)
			}
		}
	}
	if !strings.Contains(joined, "/bin/bash /dit/teardown") {
		t.Errorf("expected teardown entrypoint, got %v", args)
	}
	if !strings.Contains(joined, "DIT_IDENTITY=dit-test-id") {
		t.Errorf("expected DIT_IDENTITY env, got %v", args)
	}

	// Explicit cleanup replaces --rm: a best-effort rm before the run
	// (clears a stale container from an earlier wedged run) and a
	// guaranteed rm after it.
	first := findCall(fake.calls, dockerCmd, "rm", "-f", teardownContainer)
	last := findLastCall(fake.calls, dockerCmd, "rm", "-f", teardownContainer)
	if first < 0 || first > run {
		t.Errorf("expected `docker rm -f %s` before the run: first=%d run=%d calls=%v", teardownContainer, first, run, fake.calls)
	}
	if last < run {
		t.Errorf("expected `docker rm -f %s` after the run: last=%d run=%d calls=%v", teardownContainer, last, run, fake.calls)
	}
}

func TestDocker_TeardownDitServers_PassesOutputThroughOnSuccess(t *testing.T) {
	fake := &fakeCE{out: "Destroying pool dit-docker\nRemoving docker plugin socket\n"}
	withFakeCE(t, fake)
	d := Docker("test-id", 5001)

	out, err := d.TeardownDitServers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Destroying pool dit-docker") {
		t.Errorf("expected teardown output passed through, got %q", out)
	}
}

func TestDocker_TeardownTimeout_EnvOverride(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want time.Duration
	}{
		{"unset uses default", "", defaultTeardownTimeout},
		{"valid duration", "30s", 30 * time.Second},
		{"garbage falls back", "soon", defaultTeardownTimeout},
		{"zero falls back", "0s", defaultTeardownTimeout},
		{"negative falls back", "-1m", defaultTeardownTimeout},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("DIT_TEARDOWN_TIMEOUT", c.env)
			if got := resolveTeardownTimeout(); got != c.want {
				t.Errorf("resolveTeardownTimeout() with %q = %s, want %s", c.env, got, c.want)
			}
		})
	}
}

func TestDocker_DefaultTeardownTimeoutIsFiveMinutes(t *testing.T) {
	if defaultTeardownTimeout != 5*time.Minute {
		t.Errorf("default teardown timeout = %s, want 5m", defaultTeardownTimeout)
	}
}
