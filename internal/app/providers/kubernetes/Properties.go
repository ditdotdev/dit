// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	client "github.com/ditdotdev/dit-client-go"
)

// disablePortMappingFromRepo extracts the disablePortMapping flag from the
// kubernetes-csi metadata stored on a repository. The Kubernetes Run path
// nests its flags under a "v2" key (see metadata in kubernetes/Run.go), so a
// naive `Properties["disablePortMapping"].(bool)` panics with
// "interface {} is nil, not bool" — both Start.go and Stop.go used to.
//
// Returns false when any layer of the nested map is missing or unparseable.
func disablePortMappingFromRepo(repo client.Repository) bool {
	v2, ok := repo.Properties["v2"].(map[string]interface{})
	if !ok {
		return false
	}
	flag, _ := v2[keyDisablePortMapping].(bool)
	return flag
}
