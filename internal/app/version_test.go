package app

import (
	"testing"
)

func TestVersion_FromString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMajor int
		wantMinor int
		wantMicro int
	}{
		{
			name:      "standard version",
			input:     "1.2.3",
			wantMajor: 1,
			wantMinor: 2,
			wantMicro: 3,
		},
		{
			name:      "version with v prefix",
			input:     "v1.2.3",
			wantMajor: 1,
			wantMinor: 2,
			wantMicro: 3,
		},
		{
			name:      "zero version",
			input:     "0.0.0",
			wantMajor: 0,
			wantMinor: 0,
			wantMicro: 0,
		},
		{
			name:      "large numbers",
			input:     "10.20.30",
			wantMajor: 10,
			wantMinor: 20,
			wantMicro: 30,
		},
		{
			name:      "v prefix with zeros",
			input:     "v0.1.0",
			wantMajor: 0,
			wantMinor: 1,
			wantMicro: 0,
		},
		{
			name:      "two-part version (malformed)",
			input:     "1.2",
			wantMajor: 1,
			wantMinor: 2,
			wantMicro: 0,
		},
		{
			name:      "single number (malformed)",
			input:     "5",
			wantMajor: 5,
			wantMinor: 0,
			wantMicro: 0,
		},
		{
			name:      "v prefix two-part",
			input:     "v2.5",
			wantMajor: 2,
			wantMinor: 5,
			wantMicro: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Version{}.FromString(tt.input)
			if v.major != tt.wantMajor {
				t.Errorf("FromString(%q).major = %d, want %d", tt.input, v.major, tt.wantMajor)
			}
			if v.minor != tt.wantMinor {
				t.Errorf("FromString(%q).minor = %d, want %d", tt.input, v.minor, tt.wantMinor)
			}
			if v.micro != tt.wantMicro {
				t.Errorf("FromString(%q).micro = %d, want %d", tt.input, v.micro, tt.wantMicro)
			}
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		name string
		from Version
		to   Version
		want int
	}{
		{
			name: "equal versions",
			from: Version{1, 2, 3},
			to:   Version{1, 2, 3},
			want: 0,
		},
		{
			name: "from major greater",
			from: Version{2, 0, 0},
			to:   Version{1, 0, 0},
			want: 1,
		},
		{
			name: "from major less",
			from: Version{1, 0, 0},
			to:   Version{2, 0, 0},
			want: -1,
		},
		{
			name: "from minor greater",
			from: Version{1, 3, 0},
			to:   Version{1, 2, 0},
			want: 1,
		},
		{
			name: "from minor less",
			from: Version{1, 2, 0},
			to:   Version{1, 3, 0},
			want: -1,
		},
		{
			name: "from micro greater",
			from: Version{1, 2, 4},
			to:   Version{1, 2, 3},
			want: 1,
		},
		{
			name: "from micro less",
			from: Version{1, 2, 3},
			to:   Version{1, 2, 4},
			want: -1,
		},
		{
			name: "zero versions equal",
			from: Version{0, 0, 0},
			to:   Version{0, 0, 0},
			want: 0,
		},
		{
			name: "major dominates minor",
			from: Version{2, 0, 0},
			to:   Version{1, 99, 99},
			want: 1,
		},
		{
			name: "minor dominates micro",
			from: Version{1, 2, 0},
			to:   Version{1, 1, 99},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.from.Compare(tt.to)
			if got != tt.want {
				t.Errorf("Version{%d,%d,%d}.Compare(Version{%d,%d,%d}) = %d, want %d",
					tt.from.major, tt.from.minor, tt.from.micro,
					tt.to.major, tt.to.minor, tt.to.micro,
					got, tt.want)
			}
		})
	}
}

func TestVersion_FromString_And_Compare_Integration(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want int
	}{
		{
			name: "newer version",
			v1:   "v1.6.0",
			v2:   "v1.5.0",
			want: 1,
		},
		{
			name: "older version",
			v1:   "v1.4.0",
			v2:   "v1.5.0",
			want: -1,
		},
		{
			name: "same version",
			v1:   "v1.5.0",
			v2:   "v1.5.0",
			want: 0,
		},
		{
			name: "patch version difference",
			v1:   "1.5.1",
			v2:   "1.5.0",
			want: 1,
		},
		{
			name: "major version upgrade",
			v1:   "2.0.0",
			v2:   "1.99.99",
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from := Version{}.FromString(tt.v1)
			to := Version{}.FromString(tt.v2)
			got := from.Compare(to)
			if got != tt.want {
				t.Errorf("FromString(%q).Compare(FromString(%q)) = %d, want %d",
					tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}
