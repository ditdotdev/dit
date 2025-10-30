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
	Source      string
	Target      string
	Destination string
}

func Copy(repo string, driver string, source string, path string, port int, context string) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)
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
	r, _ := docker.GetValFromContainer(repo, "State", "Running")
	running, _ := strconv.ParseBool(r)
	if running {
		Stop(repo, port)
	}
	m, _ := docker.GetValFromContainer(repo, "HostConfig", "Mounts")
	var mounts []mount
	err = json.Unmarshal([]byte(m), &mounts)
	if err != nil {
		fmt.Printf("Failed to unmarshal mounts: %v\n", err)
		os.Exit(1)
	}
	if len(mounts) > 1 {
		fmt.Println(repo + " has more than 1 volume mount. --path is required.")
		os.Exit(1)
	}
	if path == "" {
		path = mounts[0].Target
	}
	for _, mount := range mounts {
		if mount.Target == path {
			fmt.Println("Copying data to " + mount.Source)
			v := strings.Split(mount.Source, "/")[1]
			_, _ = volumesApi.ActivateVolume(ctx, repo, v)
			vol, _, _ := volumesApi.GetVolume(ctx, repo, v)
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
			_, _ = volumesApi.DeactivateVolume(ctx, repo, v)
		}
	}
	if running {
		Start(repo, port)
	}
	fmt.Println(repo + " running with data from " + source)
}
