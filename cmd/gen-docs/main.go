// Command gen-docs regenerates the Markdown CLI reference under
// docs/src/cli/cmd/ from the live Cobra command tree. Run via `make gen-docs`.
//
// CI runs the same target and then `git diff --exit-code` against the generated
// directory — so a CLI flag or help-text change must ship with regenerated
// reference docs, or the PR fails. This keeps the public reference at
// datadatdat.com/docs/cli in sync with the actual CLI.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"datadatdat/internal/app/commands"
)

const defaultOutDir = "docs/src/cli/cmd"

// generate writes the Markdown command reference for root into outDir and
// reports progress to stdout. Pulled out of main() so it's testable.
func generate(root *cobra.Command, outDir string, stdout io.Writer) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	root.DisableAutoGenTag = true
	if err := doc.GenMarkdownTree(root, outDir); err != nil {
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
