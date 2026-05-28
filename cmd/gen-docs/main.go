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
	"os"
	"path/filepath"

	"github.com/spf13/cobra/doc"

	"datadatdat/internal/app/commands"
)

const outDir = "docs/src/cli/cmd"

func main() {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "gen-docs: %v\n", err)
		os.Exit(1)
	}

	root := commands.RootCmd()
	root.DisableAutoGenTag = true

	if err := doc.GenMarkdownTree(root, outDir); err != nil {
		fmt.Fprintf(os.Stderr, "gen-docs: %v\n", err)
		os.Exit(1)
	}

	abs, _ := filepath.Abs(outDir)
	fmt.Printf("Generated CLI reference at %s\n", abs)
}
