package local

import (
	"datadatdat/internal/app/clients"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type mount struct {
	Type        string
	Name        string
	Source      string
	Target      string
	Destination string
}

func Copy(repo string, driver string, source string, path string, port int, context string) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := clients.Docker(context, port)

	info, err := docker.InspectContainer(repo)
	if err != nil {
		fmt.Println("Container information is not available")
		os.Exit(1)
	} else {
		if info == "" {
			fmt.Println("Container information is not available")
			os.Exit(1)
		}
	}
	// Use top-level Mounts (not HostConfig.Mounts which is null for volume-driver mounts)
	m, _ := docker.GetValFromContainer(repo, "Mounts")
	var mounts []mount
	err = json.Unmarshal([]byte(m), &mounts)
	if err != nil {
		fmt.Printf("Failed to unmarshal mounts: %v\n", err)
		os.Exit(1)
	}
	r, _ := docker.GetValFromContainer(repo, "State", "Running")
	running, _ := strconv.ParseBool(r)
	if running {
		if err := Stop(repo, port); err != nil {
			fmt.Printf("Error: failed to stop container: %v\n", err)
			os.Exit(1)
		}
	}
	if path == "" {
		if len(mounts) > 1 {
			fmt.Println(repo + " has more than 1 volume mount. --destination is required.")
			os.Exit(1)
		}
		path = mounts[0].Destination
	}
	for _, mount := range mounts {
		if mount.Destination == path {
			fmt.Println("Copying data to " + mount.Source)
			// Extract volume name from Docker volume name (format: "repo_v0" -> "v0")
			parts := strings.SplitN(mount.Name, "_", 2)
			v := parts[len(parts)-1]
			_, _ = volumesApi.ActivateVolume(ctx, repo, v).Execute()
			vol, _, _ := volumesApi.GetVolume(ctx, repo, v).Execute()
			/*
				   TODO add multiple cp sources
				   when(driver) {
					   else -> docker.cp(source.removeSuffix("/"), volumeName)
				   }
			*/
			target := fmt.Sprintf("%v", vol.Config["mountpoint"])
			if _, err := docker.Cp(strings.TrimRight(source, "/"), target); err != nil {
				fmt.Printf("Warning: Failed to copy data to volume: %v\n", err)
			}
			_, _ = volumesApi.DeactivateVolume(ctx, repo, v).Execute()
		}
	}
	if running {
		if err := Start(repo, port); err != nil {
			fmt.Printf("Error: failed to start container: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println(repo + " running with data from " + source)
}
