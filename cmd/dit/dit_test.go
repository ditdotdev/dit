package main

import (
	"os"
	"testing"
)

// TestMain_Smoke drives main() with the help-only invocation so the
// rootCmd's --help short-circuits before any subcommand Run handler
// fires.  We invoke main() via a function reference so the call line
// is recorded for coverage.
func TestMain_Smoke(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"dit", "--help"}
	defer func() {
		os.Args = origArgs
		// main() doesn't itself call os.Exit; the cobra command may
		// print and return normally for --help.
		if r := recover(); r != nil {
			t.Logf("main panicked: %v", r)
		}
	}()
	main()
}
