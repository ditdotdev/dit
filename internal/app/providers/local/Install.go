// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/ditdotdev/dit/internal/app"
	"github.com/ditdotdev/dit/internal/app/utils"
	"strconv"
	"strings"
	"time"
)

var ce = utils.CommandExecutor(60, false)

func Install(latest string, registry string, verbose bool, port int, context string) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := newDockerWithRegistry(context, port, registry)

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.HideCursor = true

	fmt.Println("Initializing dit infrastructure")

	// On Windows, check WSL2 kernel compatibility before proceeding
	utils.CheckWSL2AndAdvise()

	fmt.Println("Checking docker installation")

	// Make sure Docker is running or panic
	if _, err := docker.Version(); err != nil {
		fmt.Printf("Error checking docker version: %v\n", err)
		osExit(1)
	}

	if !docker.DitLatestIsDownloaded(registry, app.Version{}.FromString(latest)) {
		var pullRegistry = registry
		if registry == "local" {
			// If local registry specified but no local image, fall back to dit
			pullRegistry = "dit"
		}
		s.Prefix = "Pulling dit docker image (may take a while) "
		s.FinalMSG = "Latest docker image downloaded"
		s.Start()
		pullImage := pullRegistry + "/dit:" + latest
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
		if _, err := docker.Remove("dit-"+context+"-server", true); err != nil {
			fmt.Printf("Warning: Failed to remove dit server: %v\n", err)
		}
		s.Stop()
	}

	launchAvailable, _ := docker.DitLaunchIsAvailable()
	if launchAvailable {
		s.Prefix = "Removing stale dit-launch container "
		s.FinalMSG = "Stale dit-launch container removed"
		s.Start()
		if _, err := docker.Remove("dit-"+context+"-launch", true); err != nil {
			fmt.Printf("Warning: Failed to remove dit launch container: %v\n", err)
		}
		s.Stop()
	}

	//TODO messages don't persist once spinner is closed

	s.Prefix = "Starting dit server docker containers "
	s.FinalMSG = "Dit CLI successfully installed, happy data versioning :)\n"
	s.Start()
	out, err := docker.LaunchDitServers()
	if err != nil {
		panic(out)
	}
	s.Stop()

	followLaunchLogs(docker, verbose)
	fmt.Println()
}

// Launch-log follow tuning; vars so tests can shrink them.
var (
	launchLogPollInterval = 200 * time.Millisecond
	launchLogTimeout      = 120 * time.Second
)

// followLaunchLogs tails the launch container's logs, echoing the banner
// (and, with verbose, everything between the START/END markers) until the
// FINISHED marker or the timeout. The previous inline loop ranged over the
// initial snapshot while appending to it - Go's range captures the slice
// length up front, so appended lines were never visited and Install returned
// before the server finished starting; the next CLI command then raced
// server startup (surfaced by context-lifecycle.bats on cold CI runners).
func followLaunchLogs(docker dockerClient, verbose bool) {
	logs := docker.FetchLaunchLogs()
	output := false
	deadline := time.Now().Add(launchLogTimeout)
	for i := 0; time.Now().Before(deadline); {
		if i >= len(logs) {
			// Caught up with the tail: refresh, or wait for more output.
			if refreshed := docker.FetchLaunchLogs(); len(refreshed) > len(logs) {
				logs = refreshed
				continue
			}
			time.Sleep(launchLogPollInterval)
			continue
		}
		line := logs[i]
		i++
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
			return
		}
	}
	fmt.Println("Warning: timed out waiting for the dit server launch to finish; it may still be starting")
}
