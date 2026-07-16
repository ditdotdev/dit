// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"fmt"
	"strconv"
)

func Remove(repo string, force bool, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	// Refuse to remove a running repository unless forced - mirrors the
	// docker provider's guard in local/Remove.go.
	if !force {
		if status, _ := k8s.GetStatefulSetStatus(repo); status == "running" {
			fmt.Println("repository " + repo + " is running, stop or use '-f' to force")
			osExit(1)
		}
	}

	// Kill any kubectl port-forward spawned by an earlier `dit run` so the
	// local port is released before we tear down the Service.
	k8s.StopPortForwarding(repo)

	k8s.DeleteStatefulSpec(repo)
	vols, _, _ := volumesApi.ListVolumes(ctx, repo).Execute()
	for _, volume := range vols {
		if _, err := volumesApi.DeleteVolume(ctx, repo, volume.Name).Execute(); err != nil {
			fmt.Printf("Warning: Failed to delete volume %s: %v\n", volume.Name, err)
		}
	}
	if _, err := repositoriesApi.DeleteRepository(ctx, repo).Execute(); err != nil {
		fmt.Printf("Error deleting repository %s: %v\n", repo, err)
		return
	}
	fmt.Println(repo + " removed")
}
