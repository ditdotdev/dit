package common

import (
	"testing"

	"github.com/datadatdat/remote-sdk-go/remote"
)

// TestParseURLResolvesAllProviderSchemes guards against the regression where a
// remote provider isn't blank-imported in RemoteAdd.go. Since the remote-sdk-go
// registry refactor, remote.ParseURL resolves a URI by asking each *registered*
// provider's FromURL; an un-imported provider isn't registered, so
// `d3 remote add <scheme>://...` fails with "no remote provider found". Every
// scheme the CLI advertises must resolve here.
func TestParseURLResolvesAllProviderSchemes(t *testing.T) {
	cases := []struct {
		name     string
		uri      string
		provider string
	}{
		{"ssh", "ssh://user@host/path", "ssh"},
		{"s3", "s3://bucket/key", "s3"},
		{"s3web", "s3web://host/path", "s3web"},
		{"nop", "nop", "nop"},
		{"datadatdat", "https://example.com/org/repo", "datadatdat"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := remote.ParseURL(tc.uri, map[string]string{})
			if err != nil {
				t.Fatalf("ParseURL(%q) returned error (provider not registered?): %v", tc.uri, err)
			}
			if res.Provider != tc.provider {
				t.Fatalf("ParseURL(%q) provider = %q, want %q", tc.uri, res.Provider, tc.provider)
			}
		})
	}
}

// TestParseURLForwardsSSHHostKeyOptions verifies the ssh provider accepts the
// skipHostCheck opt-out via -p and forwards it (camelCase) so the server can
// honor it. Regression for the strict-host-key default: without this, the CLI
// can't disable host-key checking on an ssh remote and `d3 remote add` fails.
func TestParseURLForwardsSSHHostKeyOptions(t *testing.T) {
	res, err := remote.ParseURL("ssh://user@host/path", map[string]string{"skipHostCheck": "true"})
	if err != nil {
		t.Fatalf("ParseURL with skipHostCheck returned error: %v", err)
	}
	if got, ok := res.Properties["skipHostCheck"]; !ok || got != "true" {
		t.Fatalf("skipHostCheck not forwarded: got %v (ok=%v), want \"true\"", got, ok)
	}
}
