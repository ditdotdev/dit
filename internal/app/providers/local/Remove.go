package local

import (
	"fmt"
	"os"
	"strconv"
)

func Remove(repo string, force bool, port int, context string) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := newDocker(context, port)

	// Track if we found any resources to remove
	resourcesFound := false

	id, _ := docker.GetValFromContainer(repo, "Id")
	if id != "" {
		resourcesFound = true
		if !force {
			r, _ := docker.GetValFromContainer(repo, "State", "Status")
			if r == "running" {
				fmt.Println("container " + repo + " is running, stop or use '-f' to force")
				os.Exit(1)
			}
		}
		fmt.Println("Removing repository " + repo)
		containerRunning, _ := docker.ContainerIsRunning(repo)
		if containerRunning {
			_, _ = docker.Remove(repo, force)
		} else {
			if _, err := docker.RemoveStopped(repo); err != nil {
				fmt.Printf("Warning: Failed to remove stopped container %s: %v\n", repo, err)
			}
		}
	}
	volumes, _, _ := volumesApi.ListVolumes(ctx, repo).Execute()
	if len(volumes) > 0 {
		resourcesFound = true
	}
	for _, volume := range volumes {
		fmt.Println("Deleting volume " + volume.Name)
		_, err := volumesApi.DeactivateVolume(ctx, repo, volume.Name).Execute()
		if err != nil {
			panic(err.Error())
		}
		_, err = docker.RemoveVolume(volume.Name, false)
		if err != nil {
			/*
			 * Docker will sometimes fail to launch a container after the
			 * volume has been created. The container does not exist, but
			 * docker thinks the volume is attached to a container and does
			 * not allow it to be removed. Falling back on the VolumeApi
			 * fixes this condition.
			 */
			if _, err := volumesApi.DeleteVolume(ctx, repo, volume.Name).Execute(); err != nil {
				fmt.Printf("Warning: Failed to delete volume %s: %v\n", volume.Name, err)
			}
		}
	}

	//TODO double check for docker volumes
	//vols := docker.ListVolumes(repo)
	//for _, v := range vols {
	//	vol, err := docker.RemoveVolume(v, true)
	//	if err != nil {
	//		fmt.Println(vol)
	//	}
	//}

	_, err := repositoriesApi.DeleteRepository(ctx, repo).Execute()
	if err != nil {
		// Handle 404 - repository doesn't exist
		errMsg := err.Error()
		if errMsg == "404 Not Found" {
			// If no other resources were found either, the repository doesn't exist
			if !resourcesFound {
				fmt.Printf("fatal: repository '%s' does not exist\n", repo)
				os.Exit(1)
			}
			// Otherwise, resources were removed but API record was already gone
			// Continue to success message
		} else {
			// Some other error occurred
			panic(errMsg)
		}
	} else {
		// Successfully deleted from API, mark as found
		resourcesFound = true
	}

	// Only print success if we actually found and removed something
	if resourcesFound {
		fmt.Println(repo + " removed")
	} else {
		// This shouldn't happen given the checks above, but just in case
		fmt.Printf("fatal: repository '%s' does not exist\n", repo)
		os.Exit(1)
	}
}
