package kubernetes

import (
	"fmt"
	"github.com/briandowns/spinner"
	"os"
	"strconv"
	"time"
	"datadatdat/internal/app/clients"
)

func Uninstall(force bool, removeImages bool, context string, port int) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)
	docker := clients.Docker(context, port)

	available, _ := docker.DatadatdatServerIsAvailable()
	if available {
		repos, _, _ := repositoriesApi.ListRepositories(ctx)
		for _, repo := range repos {
			if !force {
				fmt.Println("repository" + repo.Name + "exists, remove first or use '-f'")
				os.Exit(1)
			}
		}
		if _, err := docker.Remove("datadatdat-"+context+"-server", true); err != nil {
			fmt.Printf("Warning: Failed to remove datadatdat server container: %v\n", err)
		}
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.HideCursor = true

	s.Prefix = "Removing Datadatdat Docker volume "
	s.FinalMSG = "Datadatdat Docker volume removed"
	s.Start()
	if _, err := docker.RemoveVolume("datadatdat-"+context+"-date", true); err != nil {
		fmt.Printf("Warning: Failed to remove datadatdat docker volume: %v\n", err)
	}
	s.Stop()
	fmt.Println()

	if removeImages {
		s.Prefix = "Removing Datadatdat Docker image "
		s.FinalMSG = "Datadatdat Docker image removed"
		s.Start()
		if _, err := docker.RemoveDatadatdatImages("latest"); err != nil {
			fmt.Printf("Warning: Failed to remove datadatdat docker images: %v\n", err)
		}
		s.Stop()
		fmt.Println()
	}
	fmt.Println("Uninstalled datadatdat infrastructure")
}
