// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package commands

import (
	"bytes"
	"testing"
)

func resetForkFlags() {
	_ = forkCmd.Flags().Set("org", "")
	_ = forkCmd.Flags().Set("name", "")
}

func TestForkCmd_NoArgs(t *testing.T) {
	resetForkFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"fork"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("fork without args should return error")
	}
}

func TestForkCmd_Usage(t *testing.T) {
	resetForkFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"fork", "--help"})
	_ = rootCmd.Execute()
	output := buf.String()
	if output == "" {
		t.Fatal("fork --help should produce output")
	}
	if !contains(output, "Fork a remote repository") {
		t.Errorf("fork help should contain description, got: %s", output)
	}
	if !contains(output, "--org") {
		t.Errorf("fork help should show --org flag, got: %s", output)
	}
	if !contains(output, "--name") {
		t.Errorf("fork help should show --name flag, got: %s", output)
	}
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
