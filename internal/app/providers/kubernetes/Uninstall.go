package kubernetes

import (
	"fmt"
	"github.com/briandowns/spinner"
	"os"
	"strconv"
	"time"
	"titan/internal/app/clients"
)

func Uninstall(force bool, removeImages bool, context string, port int) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)
	docker := clients.Docker(context, port)

	available, _ := docker.TitanServerIsAvailable()
	if available {
		repos, _, _ := repositoriesApi.ListRepositories(ctx)
		for _, repo := range repos {
			if !force {
				fmt.Println("repository" + repo.Name + "exists, remove first or use '-f'")
				os.Exit(1)
			}
		}
		if _, err := docker.Remove("titan-"+context+"-server", true); err != nil {
			fmt.Printf("Warning: Failed to remove titan server container: %v\n", err)
		}
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.HideCursor = true

	s.Prefix = "Removing Titan Docker volume "
	s.FinalMSG = "Titan Docker volume removed"
	s.Start()
	if _, err := docker.RemoveVolume("titan-"+context+"-date", true); err != nil {
		fmt.Printf("Warning: Failed to remove titan docker volume: %v\n", err)
	}
	s.Stop()
	fmt.Println()

	if removeImages {
		s.Prefix = "Removing Titan Docker image "
		s.FinalMSG = "Titan Docker image removed"
		s.Start()
		if _, err := docker.RemoveTitanImages("latest"); err != nil {
			fmt.Printf("Warning: Failed to remove titan docker images: %v\n", err)
		}
		s.Stop()
		fmt.Println()
	}
	fmt.Println("Uninstalled titan infrastructure")
}
