package local

import (
	"fmt"
	"os"
	"strconv"
	"titan/internal/app/clients"
)

func Uninstall(version string, force bool, removeImages bool, port int, context string) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)
	docker := clients.Docker(context, port)

	serverAvailable, _ := docker.TitanServerIsAvailable()
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
	if serverAvailable {
		s, err := docker.RemoveTitanServer()
		if err != nil {
			fmt.Println(s)
			panic(err)
		}
	}
	launchAvailable, _ := docker.TitanLaunchIsAvailable()
	if launchAvailable {
		s, err := docker.RemoveTitanLaunch()
		if err != nil {
			fmt.Println(s)
			panic(err)
		}
	}

	fmt.Println("Tearing down Titan servers")
	if _, err := docker.TeardownTitanServers(); err != nil { //TODO track this
		fmt.Printf("Warning: Failed to teardown titan servers: %v\n", err)
	}

	fmt.Println("Removing titan-data Docker volume")
	if _, err := docker.RemoveTitanVolume(); err != nil { //TODO track this
		fmt.Printf("Warning: Failed to remove titan volume: %v\n", err)
	}

	if removeImages {
		fmt.Println("Removing Titan Docker image")
		if _, err := docker.RemoveTitanImages(version); err != nil { //TODO track this
			fmt.Printf("Warning: Failed to remove titan images: %v\n", err)
		}
	}
	fmt.Println("Uninstalled titan infrastructure")
}
