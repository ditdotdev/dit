package commands

import "testing"

// classifyCommand says whether a command can run without a resolved
// provider, and what default context name the install path should use.
// Regression-driven by:
//   - `d3 ls` (provider-iterating, optional)
//   - `d3 install` (creates the first context, defaults to "docker")
//   - `d3 context <anything>` (manages providers/config directly)
//   - `d3 remote ls <repo>` MUST NOT match — it needs a real provider,
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
		{"d3 ls", []string{"d3", "ls"}, true, ""},
		{"d3 ls --context foo", []string{"d3", "ls", "--context", "foo"}, true, ""},
		{"d3 install", []string{"d3", "install"}, true, "docker"},
		{"d3 context ls", []string{"d3", "context", "ls"}, true, ""},
		{"d3 context install", []string{"d3", "context", "install", "-n", "x", "-t", "kubernetes"}, true, ""},
		{"d3 context uninstall", []string{"d3", "context", "uninstall", "-f", "x"}, true, ""},
		{"d3 context default", []string{"d3", "context", "default"}, true, ""},
		// Regression: pre-fix "d3 remote ls" matched isLs because "ls" appeared
		// anywhere in args. It needs a real provider; should NOT be optional.
		{"d3 remote ls", []string{"d3", "remote", "ls", "myrepo"}, false, ""},
		{"d3 remote add", []string{"d3", "remote", "add", "uri", "repo"}, false, ""},
		{"d3 run postgres", []string{"d3", "run", "postgres:latest", "-n", "demo"}, false, ""},
		{"d3 status", []string{"d3", "status", "demo"}, false, ""},
		{"d3 rm", []string{"d3", "rm", "-f", "demo"}, false, ""},
		// Edge: empty args (shouldn't happen, but be defensive)
		{"empty", []string{}, false, ""},
		{"just binary", []string{"d3"}, false, ""},
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
