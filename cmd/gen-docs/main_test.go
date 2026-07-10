// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ditdotdev/dit/internal/app/commands"
)

// generate walks the live Cobra root and writes one .md per command. We
// point it at a temp dir to keep the test hermetic, and assert the files
// that downstream tooling depends on (dit.md as the root and dit_clone.md
// as a representative subcommand) actually show up with the expected
// flag/synopsis content.
func TestGenerate_LiveRoot(t *testing.T) {
	outDir := t.TempDir()
	var stdout bytes.Buffer

	if err := generate(commands.RootCmd(), outDir, &stdout); err != nil {
		t.Fatalf("generate: %v", err)
	}

	for _, name := range []string{"dit.md", "dit_clone.md"} {
		path := filepath.Join(outDir, name)
		data, err := os.ReadFile(path) // #nosec G304 -- test-controlled temp dir
		if err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
		if !strings.HasPrefix(string(data), "## dit") {
			t.Errorf("%s should start with '## dit', got %q", name, firstLine(string(data)))
		}
	}

	got := stdout.String()
	if !strings.Contains(got, "Generated CLI reference at ") {
		t.Errorf("stdout missing the progress line, got %q", got)
	}
}

// If a non-writable parent directory is passed, generate should report a
// clean wrapped error rather than panicking — exercises the MkdirAll
// failure branch.
func TestGenerate_MkdirError(t *testing.T) {
	// Writing inside an existing file's path forces ENOTDIR / EEXIST.
	parent := t.TempDir()
	file := filepath.Join(parent, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	bogus := filepath.Join(file, "child")

	err := generate(&cobra.Command{Use: "dit"}, bogus, new(bytes.Buffer))
	if err == nil {
		t.Fatal("expected error when outDir cannot be created")
	}
	if !strings.Contains(err.Error(), "mkdir") {
		t.Errorf("error should mention mkdir, got %v", err)
	}
}

// generate must set DisableAutoGenTag so the emitted Markdown is byte-
// stable across runs — otherwise the CI drift check (git diff
// --exit-code docs/src/cli/cmd) would fire every PR.
func TestGenerate_DeterministicOutput(t *testing.T) {
	outDir := t.TempDir()
	root := &cobra.Command{Use: "demo", Short: "demo root"}

	if err := generate(root, outDir, new(bytes.Buffer)); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !root.DisableAutoGenTag {
		t.Error("generate should set DisableAutoGenTag so output is byte-stable across runs")
	}
	if _, err := os.Stat(filepath.Join(outDir, "demo.md")); err != nil {
		t.Errorf("expected demo.md in output: %v", err)
	}
}

// run is the testable shell main() calls — exit code 0 on success,
// 1 on generate failure with the error reported to stderr.
func TestRun_Success(t *testing.T) {
	outDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := run(outDir, &stdout, &stderr); code != 0 {
		t.Errorf("run = %d, want 0; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Generated CLI reference at ") {
		t.Errorf("stdout missing progress line, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be empty on success, got %q", stderr.String())
	}
}

// On generate failure the error is reported to stderr with a gen-docs:
// prefix and the exit code is 1.
func TestRun_GenerateFails(t *testing.T) {
	parent := t.TempDir()
	file := filepath.Join(parent, "blocker")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	bogus := filepath.Join(file, "out")
	var stdout, stderr bytes.Buffer

	if code := run(bogus, &stdout, &stderr); code != 1 {
		t.Errorf("run = %d, want 1", code)
	}
	if !strings.HasPrefix(stderr.String(), "gen-docs: ") {
		t.Errorf("stderr should start with 'gen-docs: ', got %q", stderr.String())
	}
}

func firstLine(s string) string {
	before, _, _ := strings.Cut(s, "\n")
	return before
}
