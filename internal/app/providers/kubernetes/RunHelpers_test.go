// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"reflect"
	"testing"
)

func TestParseImageRef(t *testing.T) {
	cases := []struct {
		ref        string
		wantImage  string
		wantTag    string
		wantDigest string
	}{
		{"postgres", "postgres", "latest", ""},
		{"postgres:16", "postgres", "16", ""},
		{"postgres:", "postgres", "", ""},
		// Issue #240: clone hands Run the committed digest on the reference.
		{"postgres:16@sha256:abc", "postgres", "16", "sha256:abc"},
		{"postgres@sha256:abc", "postgres", "latest", "sha256:abc"},
	}
	for _, tc := range cases {
		image, tag, digest := parseImageRef(tc.ref)
		if image != tc.wantImage || tag != tc.wantTag || digest != tc.wantDigest {
			t.Errorf("parseImageRef(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tc.ref, image, tag, digest, tc.wantImage, tc.wantTag, tc.wantDigest)
		}
	}
}

func TestImageLookupRef(t *testing.T) {
	if got := imageLookupRef("postgres", "16", ""); got != "postgres:16" {
		t.Errorf("imageLookupRef without digest = %q, want postgres:16", got)
	}
	if got := imageLookupRef("postgres", "16", "sha256:abc"); got != "postgres:16@sha256:abc" {
		t.Errorf("imageLookupRef with digest = %q, want postgres:16@sha256:abc", got)
	}
}

func TestParseExposedPorts(t *testing.T) {
	meta, ports := parseExposedPorts([]string{`"5432/tcp"`, `"8080/udp:extra"`})

	wantMeta := []map[string]string{
		{"protocol": "tcp", "port": "5432"},
		{"protocol": "udp", "port": "8080"},
	}
	if !reflect.DeepEqual(meta, wantMeta) {
		t.Errorf("parseExposedPorts meta = %v, want %v", meta, wantMeta)
	}
	if want := []int{5432, 8080}; !reflect.DeepEqual(ports, want) {
		t.Errorf("parseExposedPorts ports = %v, want %v", ports, want)
	}
}
