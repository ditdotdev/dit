package kubernetes

import (
	"datadatdat/internal/app/clients"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func Uninstall(force bool, removeImages bool, context string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := clients.Docker(context, port)

	fmt.Printf("Uninstalling context '%s' (kubernetes)\n", context)

	// Track what we actually remove so the final summary reflects reality.
	// Previously this path always printed "Uninstalled datadatdat
	// infrastructure" even when nothing was removed (e.g. the container had
	// already been wiped by a prior failed uninstall).
	var removed []string

	available, _ := docker.DatadatdatServerIsAvailable()
	if available {
		repos, _, _ := repositoriesApi.ListRepositories(ctx).Execute()
		for _, repo := range repos {
			if !force {
				fmt.Println("repository '" + repo.Name + "' exists, remove first or use '-f'")
				os.Exit(1)
			}
		}
		if _, err := docker.Remove("datadatdat-"+context+"-server", true); err != nil {
			fmt.Printf("Warning: Failed to remove datadatdat server container: %v\n", err)
		} else {
			removed = append(removed, "server container")
		}
	}

	// Previously this line read "datadatdat-"+context+"-date" — a typo that
	// meant the actual per-context data volume was never removed.
	volumeName := "datadatdat-" + context + "-data"
	if docker.VolumeExists(volumeName) {
		if _, err := docker.RemoveVolume(volumeName, true); err != nil {
			fmt.Printf("Warning: Failed to remove datadatdat docker volume: %v\n", err)
		} else {
			removed = append(removed, "data volume")
		}
	}

	if removeImages {
		if _, err := docker.RemoveDatadatdatImages("latest"); err != nil {
			fmt.Printf("Warning: Failed to remove datadatdat docker images: %v\n", err)
		} else {
			removed = append(removed, "images")
		}
	}

	if len(removed) == 0 {
		fmt.Printf("No datadatdat infrastructure found for context '%s'; nothing to uninstall.\n", context)
		return
	}
	fmt.Printf("Uninstalled datadatdat infrastructure for context '%s' (%s)\n", context, strings.Join(removed, ", "))
}
