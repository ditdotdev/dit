// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	client "github.com/ditdotdev/dit-client-go"
)

// Metadata JSON keys used both here and in [Properties.go]; kept as
// constants so goconst doesn't flag the repeated string literals.
const (
	keyImage              = "image"
	keyDisablePortMapping = "disablePortMapping"
)

// splitImageTag splits a container reference into image and tag, defaulting
// the tag to "latest" when the reference has none.
func splitImageTag(container string) (string, string) {
	if !strings.Contains(container, ":") {
		return container, "latest"
	}
	parts := strings.Split(container, ":")
	return parts[0], parts[1]
}

// createDitVolumes creates a dit volume for each image volume path and
// returns the created volumes plus the volume metadata entries recorded in
// the repository properties. On a creation failure the repository is deleted
// (best effort) before panicking, matching the previous inline behavior.
func createDitVolumes(repoName string, vols []string) ([]client.Volume, []map[string]string) {
	var ditVolumes []client.Volume
	var metaVolumes []map[string]string
	for i, path := range vols {
		volName := "v" + strconv.Itoa(i)
		path := strings.Split(path, ":")[0]
		path = strings.ReplaceAll(path, `"`, "")
		fmt.Println("Creating dit volume " + volName + " with path " + path)

		v := client.Volume{
			Name:       volName,
			Properties: map[string]interface{}{"path": path},
			Config:     map[string]interface{}{},
		}
		vol, _, err := volumesApi.CreateVolume(ctx, repoName).Volume(v).Execute()
		//TODO BAD REQUEST

		if err != nil {
			if _, err := repositoriesApi.DeleteRepository(ctx, repoName).Execute(); err != nil {
				fmt.Printf("Warning: Failed to delete repository after volume creation failure: %v\n", err)
			}
			panic(err)
			//TODO REMOVE VOLUME AND EXIT
		}
		ditVolumes = append(ditVolumes, *vol)
		metaVolumes = append(metaVolumes, map[string]string{
			"name": volName,
			"path": path,
		})
	}
	return ditVolumes, metaVolumes
}

// waitForVolumesReady polls volume status until every volume reports ready,
// exiting the process if any volume reports a provisioning error.
func waitForVolumesReady(repoName string, ditVolumes []client.Volume) {
	ready := false
	for !ready {
		ready = true
		for _, v := range ditVolumes {
			s, _, _ := volumesApi.GetVolumeStatus(ctx, repoName, v.Name).Execute()
			if !s.Ready {
				ready = false
			}
			if s.GetError() != "" {
				//TODO REMOVE VOLUMES AND EXIT
				fmt.Println("Error creating volume" + v.Name + ": " + s.GetError())
				osExit(1)
			}
		}
	}
}

// parseExposedPorts converts the image's exposed-port entries into the port
// metadata recorded in the repository properties plus the numeric port list
// used for the StatefulSet and port forwarding.
func parseExposedPorts(dockerPorts []string) ([]map[string]string, []int) {
	var metaPorts []map[string]string
	ports := make([]int, 0, len(dockerPorts))
	for _, rawPort := range dockerPorts {
		rawPort = strings.ReplaceAll(rawPort, `"`, "")
		port := strings.Split(rawPort, "/")[0]
		protocol := strings.Split(strings.Split(rawPort, "/")[1], ":")[0]
		metaPorts = append(metaPorts, map[string]string{
			"protocol": protocol,
			"port":     port,
		})
		portInt, _ := strconv.Atoi(port)
		ports = append(ports, portInt)
	}
	return metaPorts, ports
}

func Run(container string, repository string, envVars []string, args []string, disablePortMap bool, privileged bool, createRepo bool, port int, context string) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := newDocker(context, port)

	if len(args) > 0 {
		fmt.Println("kubernetes provider doesn't support additional arguments")
		osExit(1)
	}

	if repository != "" && strings.Contains(repository, "/") {
		fmt.Println("Repository name cannot contain a slash")
		osExit(1)
	}

	var repoName string
	if repository == "" {
		repoName = container
	} else {
		repoName = repository
	}

	image, tag := splitImageTag(container)

	imageInfo, err := docker.InspectImage(image + ":" + tag)
	if err != nil {
		if _, err := docker.Pull(image + ":" + tag); err != nil {
			fmt.Printf("Error pulling image %s:%s: %v\n", image, tag, err)
			osExit(1)
		}
		imageInfo, _ = docker.InspectImage(image + ":" + tag)
	}
	if len(imageInfo) == 0 {
		fmt.Println("Image information is not available")
		osExit(1)
	}
	vols := docker.GetSliceFromImage(image+":"+tag, "Config", "Volumes")
	if len(vols) < 1 {
		fmt.Println("No volumes found for image " + image)
		osExit(1)
	}

	fmt.Println("Creating repository " + repoName)
	repo := client.Repository{
		Name:       repoName,
		Properties: nil,
	}
	if createRepo {
		_, _, err := repositoriesApi.CreateRepository(ctx).Repository(repo).Execute()
		if err != nil && err.Error() == "409 Conflict" {
			fmt.Println("repository '" + repo.Name + "' already exists")
			osExit(1)
		}
	}

	ditVolumes, metaVolumes := createDitVolumes(repoName, vols)

	fmt.Println("Waiting for volumes to be ready")
	waitForVolumesReady(repoName, ditVolumes)

	repoDigest := docker.GetValFromImage(image+":"+tag, "RepoDigests")
	repoDigest = strings.ReplaceAll(repoDigest, "[", "")
	repoDigest = strings.ReplaceAll(repoDigest, "]", "")
	repoDigest = strings.ReplaceAll(repoDigest, " ", "")
	repoDigest = strings.ReplaceAll(repoDigest, `"`, "")
	repoDigest = strings.TrimSpace(repoDigest)

	var imageId string
	if len(repoDigest) == 0 {
		imageId = image + ":" + tag
	} else {
		imageId = repoDigest
	}

	metadata := map[string]interface{}{
		"container": imageId,
		keyImage:    image,
		"tag":       tag,
		"digest":    repoDigest,
		"runtime":   map[string]interface{}{},
	}
	updateRepo := client.Repository{
		Name:       repoName,
		Properties: metadata,
	}
	if _, _, err := repositoriesApi.UpdateRepository(ctx, repoName).Repository(updateRepo).Execute(); err != nil {
		fmt.Printf("Warning: Failed to update repository metadata: %v\n", err)
	}

	metaPorts, ports := parseExposedPorts(docker.GetSliceFromImage(image+":"+tag, "Config", "ExposedPorts"))

	metadata = map[string]interface{}{
		"v2": map[string]interface{}{
			keyImage: map[string]interface{}{
				keyImage: image,
				"tag":    tag,
				"digest": repoDigest,
			},
			"environment":         envVars,
			"ports":               metaPorts,
			"volumes":             metaVolumes,
			keyDisablePortMapping: disablePortMap,
		},
	}

	updateRepo = client.Repository{
		Name:       repoName,
		Properties: metadata,
	}
	if _, _, err := repositoriesApi.UpdateRepository(ctx, repoName).Repository(updateRepo).Execute(); err != nil {
		fmt.Printf("Warning: Failed to update repository runtime metadata: %v\n", err)
	}

	fmt.Println("Creating " + repoName + " deployment")
	if err := k8s.CreateStatefulSet(repoName, imageId, ports, ditVolumes, envVars); err != nil {
		// Errors from CreateStatefulSet are self-describing (e.g. the
		// orphaned-resources recovery hint from issue #126 is multi-line
		// and starts with its own header). Print as-is so the formatting
		// survives.
		fmt.Fprintln(os.Stderr, err)
		osExit(1)
	}

	fmt.Println("Waiting for deployment to be ready")
	k8s.WaitForStatefulSet(repoName)

	if !disablePortMap {
		fmt.Println("Forwarding local ports")
		k8s.StartPortForwarding(repoName)
	}
}
