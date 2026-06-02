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
