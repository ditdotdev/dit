// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/ditdotdev/dit/internal/app"
	"net/http"
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
		s.FinalMSG = "Latest docker image downloaded\n"
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
		s.FinalMSG = "Old dit server removed\n"
		s.Start()
		// Containers are named after the context ("dit-<context>-server",
		// see clients.Docker getKubernetesLaunchArgs), not the context type.
		// Removing the hardcoded "dit-kubernetes-*" names left stale
		// containers behind for custom-named contexts (#214).
		if _, err := docker.Remove("dit-"+context+"-server", true); err != nil {
			fmt.Printf("Warning: Failed to remove old dit server: %v\n", err)
		}
		s.Stop()
	}

	launchAvailable, _ := docker.DitLaunchIsAvailable()
	if launchAvailable {
		s.Prefix = "Removing stale dit-launch container "
		s.FinalMSG = "Stale dit-launch container removed\n"
		s.Start()
		if _, err := docker.Remove("dit-"+context+"-launch", true); err != nil {
			fmt.Printf("Warning: Failed to remove dit-launch container: %v\n", err)
		}
		s.Stop()
	}

	s.Prefix = "Starting dit server docker containers "
	s.FinalMSG = "Dit CLI successfully installed, happy data versioning :)\n"
	s.Start()
	out, err := docker.LaunchDitKubernetesServers()
	if err != nil {
		panic(out)
	}
	s.Stop()

	followLaunchLogs(docker, verbose)
	waitForServerReady(cfg.Servers[0].URL)
	fmt.Println()
}

// Launch-log follow tuning; vars so tests can shrink them.
var (
	launchLogPollInterval = 200 * time.Millisecond
	launchLogTimeout      = 120 * time.Second
)

// Server-ready poll tuning; vars so tests can shrink or stub them.
var (
	serverReadyPollInterval = 500 * time.Millisecond
	serverReadyTimeout      = 60 * time.Second
	serverPing              = func(baseURL string) bool {
		c := http.Client{Timeout: 2 * time.Second}
		resp, err := c.Get(baseURL + "/v1/repositories")
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode < 500
	}
)

// waitForServerReady blocks until the dit server API answers (or the
// timeout passes). The launch container's FINISHED marker only means the
// launch script is done - the API server and docker-volume-proxy inside
// the server container are still starting, so a command issued right
// after install (e.g. `dit run` creating a volume through the proxy
// driver) races them. The pre-fix 120s marker timeout masked this race;
// on CI runners the gap is long enough that context-lifecycle.bats failed
// deterministically at the first post-install `dit run`.
func waitForServerReady(baseURL string) {
	deadline := time.Now().Add(serverReadyTimeout)
	for time.Now().Before(deadline) {
		if serverPing(baseURL) {
			return
		}
		time.Sleep(serverReadyPollInterval)
	}
	fmt.Println("Warning: dit server API did not respond within the startup wait; it may still be starting")
}

// followLaunchLogs tails the launch container's logs, echoing the banner
// (and, with verbose, everything between the START/END markers) until the
// FINISHED marker or the timeout. The previous inline loop ranged over the
// initial snapshot while appending to it - Go's range captures the slice
// length up front, so appended lines were never visited and Install returned
// before the server finished starting; the next CLI command then raced
// server startup (surfaced by context-lifecycle.bats on cold CI runners).
// The marker word is "DIT" (util.sh log_delimiter in dit-server); ERROR
// lines are echoed because the launch script exits fatally after emitting
// them, and the container may retry after a restart, so scanning continues
// until FINISHED or the deadline.
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
		if verbose && output && !strings.Contains(line, "DIT ") {
			fmt.Println(line)
		}
		if strings.Contains(line, "DIT START") {
			fmt.Println(strings.Replace(line, "DIT START", "", 1)[21:])
			output = true
		}
		if strings.Contains(line, "DIT END") {
			output = false
		}
		if strings.Contains(line, "DIT ERROR") {
			fmt.Println("Error: " + strings.Replace(line, "DIT ERROR", "", 1)[21:])
		}
		if strings.Contains(line, "DIT FINISHED") {
			return
		}
	}
	fmt.Println("Warning: timed out waiting for the dit server launch to finish; it may still be starting")
}
