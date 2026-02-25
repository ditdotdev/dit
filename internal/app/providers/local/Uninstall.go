package local

import (
	"datadatdat/internal/app/clients"
	"fmt"
	"os"
	"strconv"
)

func Uninstall(version string, force bool, removeImages bool, port int, context string) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)
	docker := clients.Docker(context, port)

	serverAvailable, _ := docker.DatadatdatServerIsAvailable()
	if serverAvailable {
		var repos, _, _ = repositoriesApi.ListRepositories(ctx)
		for _, repo := range repos {
			if !force {
				fmt.Println("repository '" + repo.Name + "' exists, remove first or use '-f'")
				os.Exit(1)
			}
			Remove(repo.Name, true, port, context)
		}
	}

	// Teardown ZFS pools, network, and mounts before removing containers and volumes.
	// The teardown script needs the network and volume to still be intact.
	fmt.Println("Tearing down Datadatdat servers")
	if _, err := docker.TeardownDatadatdatServers(); err != nil {
		fmt.Printf("Warning: Failed to teardown datadatdat servers: %v\n", err)
	}

	if serverAvailable {
		s, err := docker.RemoveDatadatdatServer()
		if err != nil {
			fmt.Println(s)
			panic(err)
		}
	}
	launchAvailable, _ := docker.DatadatdatLaunchIsAvailable()
	if launchAvailable {
		s, err := docker.RemoveDatadatdatLaunch()
		if err != nil {
			fmt.Println(s)
			panic(err)
		}
	}

	fmt.Println("Removing datadatdat-data Docker volume")
	if _, err := docker.RemoveDatadatdatVolume(); err != nil {
		fmt.Printf("Warning: Failed to remove datadatdat volume: %v\n", err)
	}

	if removeImages {
		fmt.Println("Removing Datadatdat Docker image")
		if _, err := docker.RemoveDatadatdatImages(version); err != nil { //TODO track this
			fmt.Printf("Warning: Failed to remove datadatdat images: %v\n", err)
		}
	}
	fmt.Println("Uninstalled datadatdat infrastructure")
}
