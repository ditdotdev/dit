package local

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func Uninstall(version string, force bool, removeImages bool, port int, context string) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := newDocker(context, port)

	fmt.Printf("Uninstalling context '%s' (docker)\n", context)

	serverAvailable, _ := docker.DatadatdatServerIsAvailable()
	if serverAvailable {
		var repos, _, _ = repositoriesApi.ListRepositories(ctx).Execute()
		for _, repo := range repos {
			if !force {
				fmt.Println("repository '" + repo.Name + "' exists, remove first or use '-f'")
				os.Exit(1)
			}
			Remove(repo.Name, true, port, context)
		}
	}

	// Track whether anything was actually removed so we can surface an
	// honest summary at the end instead of always claiming success.
	var removed []string

	// Remove both containers first to release ZFS mounts, then teardown
	// can destroy the pool and clean up network/mounts.
	if serverAvailable {
		s, err := docker.RemoveDatadatdatServer()
		if err != nil {
			fmt.Println(s)
			panic(err)
		}
		removed = append(removed, "server container")
	}
	launchAvailable, _ := docker.DatadatdatLaunchIsAvailable()
	if launchAvailable {
		s, err := docker.RemoveDatadatdatLaunch()
		if err != nil {
			fmt.Println(s)
			panic(err)
		}
		removed = append(removed, "launch container")
	}

	// Only run the ZFS teardown container when there was actually a server
	// to tear down. Running it against empty state produces misleading
	// "Tearing down Datadatdat servers" output and can waste time on hosts
	// where the teardown image has to pull.
	if serverAvailable {
		fmt.Println("Tearing down Datadatdat servers")
		if output, err := docker.TeardownDatadatdatServers(); err != nil {
			// On WSL2, the teardown container may fail to destroy the ZFS pool due to
			// hostid mismatch (pool created by WSL host, teardown runs in Docker container).
			// This is non-fatal — the pool can be cleaned up via scripts/setup-zfs-pools.sh --clean.
			fmt.Printf("Warning: Teardown encountered errors (ZFS pool may need manual cleanup via 'bash scripts/setup-zfs-pools.sh --clean'): %v\n", err)
			if output != "" {
				fmt.Println(output)
			}
		}
	}

	volumeName := "datadatdat-" + context + "-data"
	if docker.VolumeExists(volumeName) {
		if _, err := docker.RemoveDatadatdatVolume(); err != nil {
			fmt.Printf("Warning: Failed to remove datadatdat volume: %v\n", err)
		} else {
			removed = append(removed, "data volume")
		}
	}

	if removeImages {
		fmt.Println("Removing Datadatdat Docker image")
		if _, err := docker.RemoveDatadatdatImages(version); err != nil { //TODO track this
			fmt.Printf("Warning: Failed to remove datadatdat images: %v\n", err)
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
