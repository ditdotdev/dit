package kubernetes

import (
	"fmt"
	"strconv"
)

func Remove(repo string, force bool, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	// TODO check running  & force

	// Kill any kubectl port-forward spawned by an earlier `d3 run` so the
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
