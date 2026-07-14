// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout captures stdout during a function call and returns the output
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestIfContainsPrint_KeyExists(t *testing.T) {
	tests := []struct {
		name    string
		m       map[string]interface{}
		key     string
		wantKey string
		wantVal string
	}{
		{
			name:    "prints user field",
			m:       map[string]interface{}{"user": "alice"},
			key:     "user",
			wantKey: "User",
			wantVal: "alice",
		},
		{
			name:    "prints email field",
			m:       map[string]interface{}{"email": "alice@example.com"},
			key:     "email",
			wantKey: "Email",
			wantVal: "alice@example.com",
		},
		{
			name:    "prints message field",
			m:       map[string]interface{}{"message": "initial commit"},
			key:     "message",
			wantKey: "Message",
			wantVal: "initial commit",
		},
		{
			name:    "prints numeric value",
			m:       map[string]interface{}{"count": 42},
			key:     "count",
			wantKey: "Count",
			wantVal: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() {
				ifContainsPrint(tt.m, tt.key)
			})
			if !strings.Contains(output, tt.wantKey) {
				t.Errorf("ifContainsPrint() output = %q, want to contain key %q", output, tt.wantKey)
			}
			if !strings.Contains(output, tt.wantVal) {
				t.Errorf("ifContainsPrint() output = %q, want to contain value %q", output, tt.wantVal)
			}
		})
	}
}

func TestIfContainsPrint_KeyMissing(t *testing.T) {
	m := map[string]interface{}{"user": "alice"}

	output := captureStdout(func() {
		ifContainsPrint(m, "nonexistent")
	})

	if output != "" {
		t.Errorf("ifContainsPrint() with missing key should produce no output, got %q", output)
	}
}

func TestIfContainsPrint_EmptyMap(t *testing.T) {
	m := map[string]interface{}{}

	output := captureStdout(func() {
		ifContainsPrint(m, "user")
	})

	if output != "" {
		t.Errorf("ifContainsPrint() with empty map should produce no output, got %q", output)
	}
}

func TestIfContainsPrint_TitleCase(t *testing.T) {
	// Verifies the title-case logic: first letter uppercased
	m := map[string]interface{}{"source": "s3"}

	output := captureStdout(func() {
		ifContainsPrint(m, "source")
	})

	// The function capitalizes the first letter of the key
	if !strings.HasPrefix(strings.TrimSpace(output), "Source:") {
		t.Errorf("ifContainsPrint() should title-case the key, got %q", output)
	}
}

func TestRuntimeStatus_Constructor(t *testing.T) {
	tests := []struct {
		name       string
		inputName  string
		inputState string
	}{
		{
			name:       "running status",
			inputName:  "my-repo",
			inputState: "running",
		},
		{
			name:       "stopped status",
			inputName:  "test-db",
			inputState: "stopped",
		},
		{
			name:       "empty values",
			inputName:  "",
			inputState: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := RuntimeStatus(tt.inputName, tt.inputState)
			if rs.name != tt.inputName {
				t.Errorf("RuntimeStatus(%q, %q).name = %q, want %q",
					tt.inputName, tt.inputState, rs.name, tt.inputName)
			}
			if rs.status != tt.inputState {
				t.Errorf("RuntimeStatus(%q, %q).status = %q, want %q",
					tt.inputName, tt.inputState, rs.status, tt.inputState)
			}
		})
	}
}

func TestVersionConstants(t *testing.T) {
	if V1 != "v1" {
		t.Errorf("V1 constant = %q, want %q", V1, "v1")
	}
	if V2 != "v2" {
		t.Errorf("V2 constant = %q, want %q", V2, "v2")
	}
}
