// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package commands

import "testing"

// Regression for ditdotdev/dit#214: `dit context install -t kubernetes`
// without -n used to create a kubernetes context named "docker" because the
// flag default was the literal "docker" rather than the resolved type.
func TestResolveContextName(t *testing.T) {
	cases := []struct {
		name        string
		flagValue   string
		contextType string
		want        string
	}{
		{"no -n, docker type", "", "docker", "docker"},
		{"no -n, kubernetes type", "", "kubernetes", "kubernetes"},
		{"explicit -n wins over type", "minikube", "kubernetes", "minikube"},
		{"explicit -n matching type", "docker", "docker", "docker"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveContextName(tc.flagValue, tc.contextType); got != tc.want {
				t.Errorf("resolveContextName(%q, %q) = %q, want %q", tc.flagValue, tc.contextType, got, tc.want)
			}
		})
	}
}

// Guards the flag default itself: resolution in Run only happens when the
// unset flag yields "", so a reintroduced hardcoded default would silently
// bring the #214 behavior back.
func TestContextInstallNameFlagDefaultsEmpty(t *testing.T) {
	f := contextInstallCmd.Flags().Lookup(nameKey)
	if f == nil {
		t.Fatalf("context install has no --%s flag", nameKey)
	}
	if f.DefValue != "" {
		t.Errorf("--%s default = %q, want \"\" (resolved to context type at run time)", nameKey, f.DefValue)
	}
}
