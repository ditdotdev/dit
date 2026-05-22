package utils

import (
	"testing"
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
