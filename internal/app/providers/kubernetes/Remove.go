package kubernetes

import (
	"fmt"
	"strconv"
)

func Remove(repo string, force bool, port int) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)

	// TODO check running  & force
	// TODO why not working kubernetes.stopPortFowarding(repo)

	k8s.DeleteStatefulSpec(repo)
	vols, _, _ := volumesApi.ListVolumes(ctx, repo)
	for _, volume := range vols {
		if _, err := volumesApi.DeleteVolume(ctx, repo, volume.Name); err != nil {
			fmt.Printf("Warning: Failed to delete volume %s: %v\n", volume.Name, err)
		}
	}
	if _, err := repositoriesApi.DeleteRepository(ctx, repo); err != nil {
		fmt.Printf("Error deleting repository %s: %v\n", repo, err)
		return
	}
	fmt.Println(repo + " removed")
}
