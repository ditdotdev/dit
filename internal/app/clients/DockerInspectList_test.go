// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package clients

import (
	"reflect"
	"testing"
)

// Regression for #207: the inspect-list parsing removed EVERY space
// (strings.ReplaceAll(raw, " ", "")), corrupting values that legitimately
// contain one - a volume path "/data/my files" became "/data/myfiles".
// splitInspectList trims surrounding whitespace per element instead.
//
// Note the last object element loses its own closing brace: TrimRight
// strips every trailing '}' (pre-existing behavior, preserved here) -
// consumers only read the quoted key before the ':'.
func TestSplitInspectList(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{
			"object body without spaces",
			`{"/data/a":{},"/data/b":{}}`,
			[]string{`"/data/a":{}`, `"/data/b":{`},
		},
		{
			"separator whitespace trimmed, internal spaces preserved",
			`{"/data/my files":{}, "/x":{}}`,
			[]string{`"/data/my files":{}`, `"/x":{`},
		},
		{
			"array body",
			`["5432/tcp", "8080/udp"]`,
			[]string{`"5432/tcp"`, `"8080/udp"`},
		},
		{
			"EOL removed",
			"{\"a\":{}," + EOL + "\"b\":{}}",
			[]string{`"a":{}`, `"b":{`},
		},
		{
			"empty body",
			"{}",
			[]string{""},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := splitInspectList(tc.raw); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitInspectList(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}
