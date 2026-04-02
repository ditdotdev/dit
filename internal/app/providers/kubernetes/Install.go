package kubernetes

import (
	"datadatdat/internal/app"
	"datadatdat/internal/app/clients"
	"fmt"
	"github.com/briandowns/spinner"
	"os"
	"strconv"
	"strings"
	"time"
)

func Install(latest string, registry string, verbose bool, port int, context string) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)
	docker := clients.Docker(context, port)

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.HideCursor = true

	fmt.Println("Initializing datadatdat infrastructure")
	fmt.Println("Checking docker installation")

	if _, err := docker.Version(); err != nil {
		fmt.Printf("Error checking docker version: %v\n", err)
		os.Exit(1)
	}
	if !docker.DatadatdatLatestIsDownloaded(registry, app.Version{}.FromString(latest)) {
		s.Prefix = "Pulling datadatdat docker image (may take a while) "
		s.FinalMSG = "Latest docker image downloaded"
		s.Start()
		pullImage := registry + "/datadatdat:" + latest
		if _, err := docker.Pull(pullImage); err != nil {
			fmt.Printf("Error pulling image %s: %v\n", pullImage, err)
			os.Exit(1)
		}
		tagLatest := "datadatdat:" + latest
		if _, err := docker.Tag(pullImage, tagLatest); err != nil {
			fmt.Printf("Error tagging image: %v\n", err)
		}
		if _, err := docker.Tag(pullImage, "datadatdat"); err != nil {
			fmt.Printf("Error tagging image as datadatdat: %v\n", err)
		}
		s.Stop()
		fmt.Println()
	}

	serverAvailable, _ := docker.DatadatdatServerIsAvailable()
	if serverAvailable {
		s.Prefix = "Removing datadatdat server "
		s.FinalMSG = "Old datadatdat server removed"
		s.Start()
		if _, err := docker.Remove("datadatdat-kubernetes-server", true); err != nil {
			fmt.Printf("Warning: Failed to remove old datadatdat server: %v\n", err)
		}
		s.Stop()
	}

	launchAvailable, _ := docker.DatadatdatLaunchIsAvailable()
	if launchAvailable {
		s.Prefix = "Removing stale datadatdat-launch container "
		s.FinalMSG = "Stale datadatdat-launch container removed"
		s.Start()
		if _, err := docker.Remove("datadatdat-kubernetes-launch", true); err != nil {
			fmt.Printf("Warning: Failed to remove datadatdat-launch container: %v\n", err)
		}
		s.Stop()
	}

	//TODO messages don't persist once spinner is closed

	s.Prefix = "Starting datadatdat server docker containers "
	s.FinalMSG = "Datadatdat CLI successfully installed, happy data versioning :)\n"
	s.Start()
	out, err := docker.LaunchDatadatdatKubernetesServers()
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
