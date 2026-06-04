package commands

import "testing"

// RootCmd returns the package-level rootCmd so cmd/gen-docs can walk the
// command tree to regenerate the Markdown CLI reference. Asserting it
// hands back the right command keeps the gen-docs entry point honest if
// rootCmd ever gets renamed or wrapped.
func TestRootCmd(t *testing.T) {
	cmd := RootCmd()
	if cmd == nil {
		t.Fatal("RootCmd() returned nil")
	}
	if cmd.Use != "dit" {
		t.Errorf("RootCmd().Use = %q, want \"dit\"", cmd.Use)
	}
	if !cmd.HasSubCommands() {
		t.Error("RootCmd() has no subcommands; gen-docs would emit only dit.md")
	}
}

// classifyCommand says whether a command can run without a resolved
// provider, and what default context name the install path should use.
// Regression-driven by:
//   - `dit ls` (provider-iterating, optional)
//   - `dit install` (creates the first context, defaults to "docker")
//   - `dit context <anything>` (manages providers/config directly)
//   - `dit remote ls <repo>` MUST NOT match — it needs a real provider,
//     but pre-fix the matching scanned the whole args slice for "ls"
//     and treated this case as optional, leaving provider==nil and
//     panicking with SIGSEGV at remote.go:45.
func TestClassifyCommand(t *testing.T) {
	cases := []struct {
		name         string
		args         []string
		wantOptional bool
		wantDefault  string
	}{
		{"dit ls", []string{"dit", "ls"}, true, ""},
		{"dit ls --context foo", []string{"dit", "ls", "--context", "foo"}, true, ""},
		{"dit install", []string{"dit", "install"}, true, "docker"},
		{"dit context ls", []string{"dit", "context", "ls"}, true, ""},
		{"dit context install", []string{"dit", "context", "install", "-n", "x", "-t", "kubernetes"}, true, ""},
		{"dit context uninstall", []string{"dit", "context", "uninstall", "-f", "x"}, true, ""},
		{"dit context default", []string{"dit", "context", "default"}, true, ""},
		// Regression: pre-fix "dit remote ls" matched isLs because "ls" appeared
		// anywhere in args. It needs a real provider; should NOT be optional.
		{"dit remote ls", []string{"dit", "remote", "ls", "myrepo"}, false, ""},
		{"dit remote add", []string{"dit", "remote", "add", "uri", "repo"}, false, ""},
		{"dit run postgres", []string{"dit", "run", "postgres:latest", "-n", "demo"}, false, ""},
		{"dit status", []string{"dit", "status", "demo"}, false, ""},
		{"dit rm", []string{"dit", "rm", "-f", "demo"}, false, ""},
		// Edge: empty args (shouldn't happen, but be defensive)
		{"empty", []string{}, false, ""},
		{"just binary", []string{"dit"}, false, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotOptional, gotDefault := classifyCommand(c.args)
			if gotOptional != c.wantOptional {
				t.Errorf("optional = %v, want %v", gotOptional, c.wantOptional)
			}
			if gotDefault != c.wantDefault {
				t.Errorf("default = %q, want %q", gotDefault, c.wantDefault)
			}
		})
	}
}
