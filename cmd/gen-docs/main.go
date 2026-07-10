// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

// Command gen-docs regenerates the Markdown CLI reference under
// docs/src/cli/cmd/ from the live Cobra command tree. Run via `make gen-docs`.
//
// CI runs the same target and then `git diff --exit-code` against the generated
// directory — so a CLI flag or help-text change must ship with regenerated
// reference docs, or the PR fails. This keeps the public reference at
// dit.dev/docs/cli in sync with the actual CLI.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/ditdotdev/dit/internal/app/commands"
)

const defaultOutDir = "docs/src/cli/cmd"

// generate writes the Markdown command reference for root into outDir and
// reports progress to stdout. Pulled out of main() so it's testable.
//
// Uses GenMarkdownTreeCustom so the SEE ALSO cross-reference links land
// as `[dit](dit)` instead of `[dit](dit.md)` — the Next.js renderer at
// dit.dev/docs routes by clean slug (/docs/cli/cmd/dit), and the
// default `.md` suffix 404s.
func generate(root *cobra.Command, outDir string, stdout io.Writer) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	root.DisableAutoGenTag = true
	emptyPrepender := func(string) string { return "" }
	stripMdSuffix := func(link string) string { return strings.TrimSuffix(link, ".md") }
	if err := doc.GenMarkdownTreeCustom(root, outDir, emptyPrepender, stripMdSuffix); err != nil {
		return fmt.Errorf("gen markdown tree: %w", err)
	}
	abs, _ := filepath.Abs(outDir)
	_, _ = fmt.Fprintf(stdout, "Generated CLI reference at %s\n", abs)
	return nil
}

// run is the testable shell for main(): it isolates the side effects
// (resolving the live root command, writing to stdout/stderr, exit code)
// behind a single int-returning function so main() is a one-liner that
// CI doesn't need to cover.
func run(outDir string, stdout, stderr io.Writer) int {
	if err := generate(commands.RootCmd(), outDir, stdout); err != nil {
		_, _ = fmt.Fprintf(stderr, "gen-docs: %v\n", err)
		return 1
	}
	return 0
}

func main() { os.Exit(run(defaultOutDir, os.Stdout, os.Stderr)) }
