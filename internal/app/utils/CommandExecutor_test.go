// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestHelperProcess is not a real test: it is re-executed as a child
// process by the ExecWithTimeout tests (the standard os/exec testing
// pattern), so the executor is exercised against a real subprocess on
// every platform without depending on `sleep`/`timeout` binaries.
//
// Modes (first arg after "--"):
//
//	hang   - print a line, then sleep far longer than any test timeout
//	ok     - print a line and exit 0
//	fail   - print a line and exit 3
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	mode := ""
	if len(args) > 0 {
		mode = args[0]
	}
	switch mode {
	case "hang":
		fmt.Println("Removing docker plugin socket")
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case "ok":
		fmt.Println("helper ok")
		os.Exit(0)
	case "fail":
		fmt.Println("helper failed")
		os.Exit(3)
	default:
		fmt.Fprintf(os.Stderr, "unknown helper mode %q\n", mode)
		os.Exit(2)
	}
}

// helperArgs builds the argv that re-runs this test binary as the helper.
func helperArgs(mode string) (string, []string) {
	return os.Args[0], []string{"-test.run=^TestHelperProcess$", "--", mode}
}

func withHelperEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
}

func TestExecWithTimeout_KillsHungProcess(t *testing.T) {
	withHelperEnv(t)
	ce := CommandExecutor(60, false)
	name, args := helperArgs("hang")

	// The deadline must comfortably exceed a cold process start (a freshly
	// built test binary can take well over half a second to launch on
	// Windows) so the helper gets to print before it is killed, while
	// staying far below the helper's 30s sleep.
	const deadline = 2 * time.Second
	start := time.Now()
	out, err := ce.ExecWithTimeout(deadline, name, args...)
	elapsed := time.Since(start)

	if elapsed > 10*time.Second {
		t.Fatalf("ExecWithTimeout took %s; a hung child must be abandoned at the deadline", elapsed)
	}
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error should wrap context.DeadlineExceeded, got %v", err)
	}
	if !strings.Contains(out, "Removing docker plugin socket") {
		t.Errorf("output produced before the hang must be returned, got %q", out)
	}
}

func TestExecWithTimeout_SuccessPassesOutput(t *testing.T) {
	withHelperEnv(t)
	ce := CommandExecutor(60, false)
	name, args := helperArgs("ok")

	out, err := ce.ExecWithTimeout(10*time.Second, name, args...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "helper ok") {
		t.Errorf("expected helper output, got %q", out)
	}
}

func TestExecWithTimeout_NonZeroExitFoldsOutputIntoError(t *testing.T) {
	withHelperEnv(t)
	ce := CommandExecutor(60, false)
	name, args := helperArgs("fail")

	out, err := ce.ExecWithTimeout(10*time.Second, name, args...)
	if err == nil {
		t.Fatal("expected an error for a non-zero exit")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a plain failure must not look like a timeout: %v", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Errorf("error should wrap *exec.ExitError, got %T %v", err, err)
	}
	// Same contract as Exec: the command's own output is folded into the
	// error so callers that only print err still show the diagnosis.
	if !strings.Contains(err.Error(), "helper failed") {
		t.Errorf("error should include command output, got %v", err)
	}
	if !strings.Contains(out, "helper failed") {
		t.Errorf("expected output returned, got %q", out)
	}
}

func TestExec_NonZeroExitFoldsOutputIntoError(t *testing.T) {
	withHelperEnv(t)
	ce := CommandExecutor(60, false)
	name, args := helperArgs("fail")

	_, err := ce.Exec(name, args...)
	if err == nil {
		t.Fatal("expected an error for a non-zero exit")
	}
	if !strings.Contains(err.Error(), "helper failed") {
		t.Errorf("error should include command output, got %v", err)
	}
}
