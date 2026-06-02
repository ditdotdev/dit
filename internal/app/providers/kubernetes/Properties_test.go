package kubernetes

import (
	"testing"

	client "github.com/ditdotdev/dit-client-go"
)

// disablePortMappingFromRepo must read the flag from the nested v2 metadata
// produced by the kubernetes Run path. Pre-fix Stop.go and Start.go used
// `repo.Properties["disablePortMapping"].(bool)` which panicked because the
// real shape is `Properties["v2"]["disablePortMapping"]`.
func TestDisablePortMappingFromRepo(t *testing.T) {
	cases := []struct {
		name string
		repo client.Repository
		want bool
	}{
		{
			name: "nil properties",
			repo: client.Repository{Name: "r"},
			want: false,
		},
		{
			name: "no v2 key",
			repo: client.Repository{
				Name:       "r",
				Properties: map[string]interface{}{"other": 1},
			},
			want: false,
		},
		{
			name: "v2 present but no disablePortMapping",
			repo: client.Repository{
				Name:       "r",
				Properties: map[string]interface{}{"v2": map[string]interface{}{"image": "postgres"}},
			},
			want: false,
		},
		{
			name: "v2.disablePortMapping=false",
			repo: client.Repository{
				Name:       "r",
				Properties: map[string]interface{}{"v2": map[string]interface{}{"disablePortMapping": false}},
			},
			want: false,
		},
		{
			name: "v2.disablePortMapping=true",
			repo: client.Repository{
				Name:       "r",
				Properties: map[string]interface{}{"v2": map[string]interface{}{"disablePortMapping": true}},
			},
			want: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := disablePortMappingFromRepo(c.repo); got != c.want {
				t.Errorf("disablePortMappingFromRepo(%s) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}
