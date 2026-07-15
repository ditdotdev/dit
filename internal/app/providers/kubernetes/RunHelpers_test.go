// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

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
