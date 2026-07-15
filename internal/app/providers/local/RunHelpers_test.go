// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"reflect"
	"testing"
)

func TestSplitImageTag(t *testing.T) {
	cases := []struct {
		container string
		wantImage string
		wantTag   string
	}{
		{"postgres", "postgres", "latest"},
		{"postgres:16", "postgres", "16"},
		{"postgres:", "postgres", ""},
	}
	for _, tc := range cases {
		image, tag := splitImageTag(tc.container)
		if image != tc.wantImage || tag != tc.wantTag {
			t.Errorf("splitImageTag(%q) = (%q, %q), want (%q, %q)",
				tc.container, image, tag, tc.wantImage, tc.wantTag)
		}
	}
}

func TestFilterRunArgs(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		imageTag string
		want     []string
	}{
		{"strips --name and value", []string{flagName, "custom", "-e", "X=1"}, "img:latest", []string{"-e", "X=1"}},
		{"strips trailing --name without value", []string{"-e", "X=1", flagName}, "img:latest", []string{"-e", "X=1"}},
		{"strips image:tag token", []string{"-e", "X=1", "img:latest"}, "img:latest", []string{"-e", "X=1"}},
		{"keeps everything else", []string{"-p", "8080:80"}, "img:latest", []string{"-p", "8080:80"}},
		{"empty args", nil, "img:latest", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := filterRunArgs(tc.args, tc.imageTag); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("filterRunArgs(%v, %q) = %v, want %v", tc.args, tc.imageTag, got, tc.want)
			}
		})
	}
}

func TestPortRunArgs(t *testing.T) {
	raw := []string{`"5432/tcp"`, `"8080/udp:extra"`, `malformed`}

	t.Run("port mapping enabled", func(t *testing.T) {
		args, meta := portRunArgs(raw, false)
		wantArgs := []string{"-p", "5432:5432/tcp", "-p", "8080:8080/udp"}
		if !reflect.DeepEqual(args, wantArgs) {
			t.Errorf("portRunArgs args = %v, want %v", args, wantArgs)
		}
		wantMeta := []map[string]string{
			{"protocol": "tcp", "port": "5432"},
			{"protocol": "udp", "port": "8080"},
		}
		if !reflect.DeepEqual(meta, wantMeta) {
			t.Errorf("portRunArgs meta = %v, want %v", meta, wantMeta)
		}
	})

	t.Run("port mapping disabled still records metadata", func(t *testing.T) {
		args, meta := portRunArgs(raw, true)
		if args != nil {
			t.Errorf("portRunArgs args = %v, want none", args)
		}
		if len(meta) != 2 {
			t.Errorf("portRunArgs meta has %d entries, want 2", len(meta))
		}
	})
}

func TestFirstRepoDigest(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"single digest", "[\"postgres@sha256:abc\"]\n", "postgres@sha256:abc"},
		{"multiple digests takes first", `["a@sha256:1", "b@sha256:2"]`, "a@sha256:1"},
		{"empty", "[]", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := firstRepoDigest(tc.raw); got != tc.want {
				t.Errorf("firstRepoDigest(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
