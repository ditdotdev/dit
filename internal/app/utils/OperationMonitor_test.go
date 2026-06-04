package utils

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	ditclient "github.com/ditdotdev/dit-client-go"
)

func TestOperationMonitor_IsTerminal(t *testing.T) {
	om := operationMonitor{}

	tests := []struct {
		name  string
		state string
		want  bool
	}{
		{
			name:  "COMPLETE is terminal",
			state: OperationStateComplete,
			want:  true,
		},
		{
			name:  "FAILED is terminal",
			state: OperationStateFailed,
			want:  true,
		},
		{
			name:  "ABORT is terminal",
			state: OperationStateAbort,
			want:  true,
		},
		{
			name:  "START is not terminal",
			state: "START",
			want:  false,
		},
		{
			name:  "PROGRESS is not terminal",
			state: "PROGRESS",
			want:  false,
		},
		{
			name:  "empty string is not terminal",
			state: "",
			want:  false,
		},
		{
			name:  "lowercase complete is not terminal",
			state: "complete",
			want:  false,
		},
		{
			name:  "RUNNING is not terminal",
			state: "RUNNING",
			want:  false,
		},
		{
			name:  "PENDING is not terminal",
			state: "PENDING",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := om.IsTerminal(tt.state)
			if got != tt.want {
				t.Errorf("IsTerminal(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestFormatProgressLine(t *testing.T) {
	tests := []struct {
		name   string
		msg    string
		padLen int
		want   string
	}{
		{
			name:   "msg equal to padLen",
			msg:    "hello",
			padLen: 5,
			want:   "\rhello",
		},
		{
			name:   "msg shorter than padLen pads with trailing spaces",
			msg:    "hi",
			padLen: 5,
			want:   "\rhi   ",
		},
		{
			name:   "msg longer than padLen is not truncated",
			msg:    "hello world",
			padLen: 5,
			want:   "\rhello world",
		},
		{
			name:   "empty msg with padLen",
			msg:    "",
			padLen: 3,
			want:   "\r   ",
		},
		{
			name:   "padLen zero",
			msg:    "x",
			padLen: 0,
			want:   "\rx",
		},
		// Regression: the previous implementation did
		// msg[0:(padLen-len(msg)+1)] which panicked with a slice
		// out-of-range when msg was shorter than padLen. Exercise
		// that exact case to make sure the new helper does not panic.
		{
			name:   "shorter follow-up message after long message does not panic",
			msg:    "ab",
			padLen: 20,
			want:   "\rab                  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatProgressLine(tt.msg, tt.padLen)
			if got != tt.want {
				t.Errorf("formatProgressLine(%q, %d) = %q, want %q",
					tt.msg, tt.padLen, got, tt.want)
			}
		})
	}
}

func TestOperationStateConstants(t *testing.T) {
	// Verify the constant values haven't accidentally changed
	if OperationStateComplete != "COMPLETE" {
		t.Errorf("OperationStateComplete = %q, want %q", OperationStateComplete, "COMPLETE")
	}
	if OperationStateFailed != "FAILED" {
		t.Errorf("OperationStateFailed = %q, want %q", OperationStateFailed, "FAILED")
	}
	if OperationStateAbort != "ABORT" {
		t.Errorf("OperationStateAbort = %q, want %q", OperationStateAbort, "ABORT")
	}
}

func TestCommandExecutor_Constructor(t *testing.T) {
	tests := []struct {
		name        string
		timeout     int
		debug       bool
		wantTimeout int
		wantDebug   bool
	}{
		{
			name:        "default timeout when zero",
			timeout:     0,
			debug:       false,
			wantTimeout: 60,
			wantDebug:   false,
		},
		{
			name:        "custom timeout",
			timeout:     120,
			debug:       false,
			wantTimeout: 120,
			wantDebug:   false,
		},
		{
			name:        "negative timeout gets default",
			timeout:     -1,
			debug:       false,
			wantTimeout: 60,
			wantDebug:   false,
		},
		{
			name:        "debug enabled",
			timeout:     30,
			debug:       true,
			wantTimeout: 30,
			wantDebug:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ce := CommandExecutor(tt.timeout, tt.debug)
			if ce.timeout != tt.wantTimeout {
				t.Errorf("CommandExecutor(%d, %v).timeout = %d, want %d",
					tt.timeout, tt.debug, ce.timeout, tt.wantTimeout)
			}
			if ce.debug != tt.wantDebug {
				t.Errorf("CommandExecutor(%d, %v).debug = %v, want %v",
					tt.timeout, tt.debug, ce.debug, tt.wantDebug)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// OperationMonitor constructor + Monitor() loop
// ---------------------------------------------------------------------------

func TestOperationMonitor_Constructor(t *testing.T) {
	op := ditclient.Operation{Id: "op-1", Type: "PUSH"}
	om := OperationMonitor("myrepo", op)
	if om.repo != "myrepo" {
		t.Errorf("repo = %q, want myrepo", om.repo)
	}
	if om.operation.Id != "op-1" {
		t.Errorf("operation.Id = %q, want op-1", om.operation.Id)
	}
}

// startMonitorMock returns a mock progress endpoint that returns the
// supplied JSON body for every progress GET. Returns the port number.
func startMonitorMock(t *testing.T, body string) int {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	p, _ := strconv.Atoi(u.Port())
	return p
}

// captureStdout runs fn while redirecting stdout, then returns what was
// printed. Used to confirm Monitor's final status message.
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// shortPolling overrides the polling/idle settings for fast tests. Restores
// the originals on cleanup.
func shortPolling(t *testing.T) {
	t.Helper()
	origPoll := MonitorPollInterval
	origIdle := MonitorIdleTimeout
	MonitorPollInterval = 1 * time.Millisecond
	MonitorIdleTimeout = 50 * time.Millisecond
	t.Cleanup(func() {
		MonitorPollInterval = origPoll
		MonitorIdleTimeout = origIdle
	})
}

func TestMonitor_CompleteState_PushReturnsTrue(t *testing.T) {
	shortPolling(t)
	// Server returns a single entry with COMPLETE type, which is terminal.
	port := startMonitorMock(t, `[{"id":1,"type":"COMPLETE","message":"all done"}]`)

	om := OperationMonitor("repo", ditclient.Operation{Id: "op-x", Type: "PUSH"})
	var ok bool
	out := captureStdout(func() { ok = om.Monitor(port) })

	if !ok {
		t.Errorf("Monitor with COMPLETE should return true")
	}
	if !strings.Contains(out, "Push completed successfully") {
		t.Errorf("expected completion message, got %q", out)
	}
}

func TestMonitor_FailedState_PullReturnsFalse(t *testing.T) {
	shortPolling(t)
	port := startMonitorMock(t, `[{"id":1,"type":"FAILED","message":"bad"}]`)

	om := OperationMonitor("repo", ditclient.Operation{Id: "op-x", Type: "PULL"})
	var ok bool
	out := captureStdout(func() { ok = om.Monitor(port) })

	if ok {
		t.Errorf("Monitor with FAILED should return false")
	}
	if !strings.Contains(out, "Pull failed") {
		t.Errorf("expected failed message, got %q", out)
	}
}

func TestMonitor_AbortState_PrintsAbortedMessage(t *testing.T) {
	shortPolling(t)
	port := startMonitorMock(t, `[{"id":1,"type":"ABORT","message":"by user"}]`)

	om := OperationMonitor("repo", ditclient.Operation{Id: "op-x", Type: "PUSH"})
	out := captureStdout(func() { om.Monitor(port) })

	if !strings.Contains(out, "Push aborted") {
		t.Errorf("expected aborted message, got %q", out)
	}
}

func TestMonitor_ProgressThenComplete_PrintsProgressLines(t *testing.T) {
	shortPolling(t)
	// First request returns PROGRESS then COMPLETE so the for-loop walks
	// the entries slice and exercises both branches (PROGRESS uses pad,
	// non-PROGRESS prints message).
	port := startMonitorMock(t, `[
		{"id":1,"type":"PROGRESS","message":"step 1"},
		{"id":2,"type":"MESSAGE","message":"info"},
		{"id":3,"type":"COMPLETE","message":"done"}
	]`)
	om := OperationMonitor("repo", ditclient.Operation{Id: "op-x", Type: "PUSH"})
	out := captureStdout(func() { om.Monitor(port) })
	if !strings.Contains(out, "step 1") {
		t.Errorf("expected progress line in output, got %q", out)
	}
	if !strings.Contains(out, "info") {
		t.Errorf("expected MESSAGE line in output, got %q", out)
	}
}

func TestMonitor_HTTPError_BreaksLoop(t *testing.T) {
	shortPolling(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())

	om := OperationMonitor("repo", ditclient.Operation{Id: "op-x", Type: "PUSH"})
	var ok bool
	out := captureStdout(func() { ok = om.Monitor(port) })

	if ok {
		t.Errorf("Monitor on HTTP error should return false")
	}
	if !strings.Contains(out, "Error monitoring operation") {
		t.Errorf("expected error message, got %q", out)
	}
}

func TestMonitor_IdleTimeout_BreaksLoop(t *testing.T) {
	shortPolling(t)
	// Empty entry list keeps the loop running until MonitorIdleTimeout
	// (50ms above) elapses without progress, which prints "No progress
	// from server" and breaks.
	port := startMonitorMock(t, `[]`)
	om := OperationMonitor("repo", ditclient.Operation{Id: "op-x", Type: "PUSH"})
	out := captureStdout(func() { om.Monitor(port) })
	if !strings.Contains(out, "No progress from server") {
		t.Errorf("expected idle-timeout message, got %q", out)
	}
}

// ---------------------------------------------------------------------------
// CommandExecutor.Exec — exercise the real shell-out path with `go`
// (which is guaranteed present in any test environment).
// ---------------------------------------------------------------------------

func TestCommandExecutor_Exec_RunsCommand(t *testing.T) {
	ce := CommandExecutor(5, false)
	// `go version` is portable and reliably exits 0.
	out, err := ce.Exec("go", "version")
	if err != nil {
		t.Skipf("`go version` failed in test env: %v", err)
	}
	if !strings.Contains(out, "go") {
		t.Errorf("expected `go version` output to contain 'go', got %q", out)
	}
}

func TestCommandExecutor_Exec_NonExistentCommand_Errors(t *testing.T) {
	ce := CommandExecutor(5, false)
	_, err := ce.Exec("definitely-not-a-real-binary-xyz")
	if err == nil {
		t.Error("expected error for non-existent binary")
	}
}

func TestCommandExecutor_Exec_WithDebug_NoOutputOnFailureForTest(t *testing.T) {
	ce := CommandExecutor(5, true)
	// Debug branch prints to stdout — we just want to exercise it.
	_, err := ce.Exec("definitely-not-a-real-binary-xyz-2")
	if err == nil {
		t.Error("expected error for non-existent binary")
	}
}

func TestCommandExecutor_SetDebug(t *testing.T) {
	ce := CommandExecutor(60, false)

	if ce.debug != false {
		t.Fatal("initial debug should be false")
	}

	ce.SetDebug(true)
	if ce.debug != true {
		t.Error("SetDebug(true) should set debug to true")
	}

	ce.SetDebug(false)
	if ce.debug != false {
		t.Error("SetDebug(false) should set debug to false")
	}
}
