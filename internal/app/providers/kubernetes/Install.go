// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/ditdotdev/dit/internal/app"
	"os"
	"strconv"
	"strings"
	"time"
)

func Install(latest string, registry string, verbose bool, port int, context string, properties []string) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := newDocker(context, port)

	// Context-specific parameters (`-p key=value`) are handed to the server as
	// DIT_CONTEXT_CONFIG: a comma-separated key=value string the server parses
	// into its context configuration — e.g. storageClass / snapshotClass for the
	// CSI driver (see dit-server Application.kt). LaunchDitKubernetesServers
	// forwards DIT_* env vars into the server container, so exporting it here
	// plumbs the params through to the running context. Without this the params
	// are silently dropped and dit-provisioned PVCs fall back to the cluster
	// default StorageClass, which is often non-CSI and cannot snapshot.
	if len(properties) > 0 {
		for _, p := range properties {
			if strings.Count(p, "=") != 1 || strings.HasPrefix(p, "=") || strings.HasSuffix(p, "=") {
				fmt.Printf("Error: invalid context parameter '%s' (expected key=value)\n", p)
				osExit(1)
				return
			}
		}
		// Name is a fixed, valid identifier, so Setenv cannot fail here.
		_ = os.Setenv("DIT_CONTEXT_CONFIG", strings.Join(properties, ","))
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.HideCursor = true

	fmt.Println("Initializing dit infrastructure")
	fmt.Println("Checking docker installation")

	if _, err := docker.Version(); err != nil {
		fmt.Printf("Error checking docker version: %v\n", err)
		osExit(1)
	}
	if !docker.DitLatestIsDownloaded(registry, app.Version{}.FromString(latest)) {
		s.Prefix = "Pulling dit docker image (may take a while) "
		s.FinalMSG = "Latest docker image downloaded"
		s.Start()
		pullImage := registry + "/dit:" + latest
		if _, err := docker.Pull(pullImage); err != nil {
			fmt.Printf("Error pulling image %s: %v\n", pullImage, err)
			osExit(1)
		}
		tagLatest := "dit:" + latest
		if _, err := docker.Tag(pullImage, tagLatest); err != nil {
			fmt.Printf("Error tagging image: %v\n", err)
		}
		if _, err := docker.Tag(pullImage, "dit"); err != nil {
			fmt.Printf("Error tagging image as dit: %v\n", err)
		}
		s.Stop()
		fmt.Println()
	}

	serverAvailable, _ := docker.DitServerIsAvailable()
	if serverAvailable {
		s.Prefix = "Removing dit server "
		s.FinalMSG = "Old dit server removed"
		s.Start()
		if _, err := docker.Remove("dit-kubernetes-server", true); err != nil {
			fmt.Printf("Warning: Failed to remove old dit server: %v\n", err)
		}
		s.Stop()
	}

	launchAvailable, _ := docker.DitLaunchIsAvailable()
	if launchAvailable {
		s.Prefix = "Removing stale dit-launch container "
		s.FinalMSG = "Stale dit-launch container removed"
		s.Start()
		if _, err := docker.Remove("dit-kubernetes-launch", true); err != nil {
			fmt.Printf("Warning: Failed to remove dit-launch container: %v\n", err)
		}
		s.Stop()
	}

	//TODO messages don't persist once spinner is closed

	s.Prefix = "Starting dit server docker containers "
	s.FinalMSG = "Dit CLI successfully installed, happy data versioning :)\n"
	s.Start()
	out, err := docker.LaunchDitKubernetesServers()
	if err != nil {
		panic(out)
	}
	s.Stop()

	output := false
	logs := docker.FetchLaunchLogs()
	for _, line := range logs {
		if verbose && output && !strings.Contains(line, "DATADATDAT") {
			fmt.Println(line)
		}
		if strings.Contains(line, "DATADATDAT START") {
			fmt.Println(strings.Replace(line, "DATADATDAT START", "", 1)[21:])
			output = true
		}
		if strings.Contains(line, "DATADATDAT END") {
			output = false
		}
		if strings.Contains(line, "DATADATDAT FINISHED") {
			break
		}
		newLogs := docker.FetchLaunchLogs()
		if len(newLogs) > len(logs) {
			logs = append(logs, newLogs[len(logs):]...)
		}
	}
	fmt.Println()
}
