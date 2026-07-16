// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"fmt"
	"strconv"
	"strings"

	client "github.com/ditdotdev/dit-client-go"
)

// Port-metadata map keys recorded in the repository properties; kept as
// constants so goconst doesn't flag the repeated string literals.
const (
	keyProtocol = "protocol"
	keyPort     = "port"
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

// createRepoVolumes creates one docker volume per image volume path and
// returns the `-v name:path` docker arguments plus the volume metadata
// entries recorded in the repository properties.
func createRepoVolumes(docker dockerClient, containerName string, vols []string) ([]string, []map[string]string, error) {
	var volArgs []string
	var metaVols []map[string]string
	for i, path := range vols {
		volumeName := "v" + strconv.Itoa(i)
		volName := docker.FormatVolumeName(containerName, volumeName)
		path := strings.Split(path, ":")[0]
		path = strings.ReplaceAll(path, `"`, "")

		fmt.Println("Creating docker volume " + volName + " with path " + path)
		if _, err := docker.CreateVolume(volName, path); err != nil {
			return nil, nil, err
		}
		volArgs = append(volArgs, "-v", volName+":"+path)
		metaVols = append(metaVols, map[string]string{"name": volumeName, "path": path})
	}
	return volArgs, metaVols, nil
}

// filterRunArgs strips the --name flag (and its value) and the image:tag
// token from the pass-through docker args - Run supplies its own name and
// image.
func filterRunArgs(args []string, imageTag string) []string {
	var filtered []string
	i := 0
	for i < len(args) {
		switch args[i] {
		case flagName:
			// Skip --name and the next argument
			if i+1 < len(args) {
				i += 2 // Skip both --name and its value
			} else {
				i += 1 // Just skip --name if no value follows
			}
		case imageTag:
			// Skip the image:tag argument
			i += 1
		default:
			// Keep this argument
			filtered = append(filtered, args[i])
			i += 1
		}
	}
	return filtered
}

// portRunArgs parses the image's exposed-port entries into `-p` publish
// arguments (omitted when port mapping is disabled) and the port metadata
// entries recorded in the repository properties. Malformed entries are
// skipped.
func portRunArgs(rawPorts []string, disablePortMap bool) ([]string, []map[string]string) {
	var portArgs []string
	var metaPorts []map[string]string
	for _, rawPort := range rawPorts {
		rawPort = strings.ReplaceAll(rawPort, `"`, "")
		portParts := strings.Split(rawPort, "/")
		if len(portParts) < 2 {
			continue // Skip malformed port entries
		}
		port := portParts[0]
		protocol := strings.Split(portParts[1], ":")[0]
		if !disablePortMap {
			portArgs = append(portArgs, "-p", port+":"+port+"/"+protocol)
		}
		metaPorts = append(metaPorts, map[string]string{keyProtocol: protocol, keyPort: port})
	}
	return portArgs, metaPorts
}

// firstRepoDigest normalizes the raw RepoDigests value from image metadata
// and returns the first digest, or "" when the image has none.
func firstRepoDigest(raw string) string {
	digest := strings.ReplaceAll(raw, "[", "")
	digest = strings.ReplaceAll(digest, "]", "")
	digest = strings.ReplaceAll(digest, " ", "")
	digest = strings.ReplaceAll(digest, `"`, "")
	digest = strings.ReplaceAll(digest, "\n", "")
	digest = strings.TrimSpace(digest)

	// If multiple digests are present (separated by commas), take the first one
	if strings.Contains(digest, ",") {
		digest = strings.TrimSpace(strings.Split(digest, ",")[0])
	}
	return digest
}

func Run(container string, repository string, envVars []string, args []string, disablePortMap bool, privileged bool, createRepo bool, port int, context string) (string, error) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := newDocker(context, port)

	// Validate repository name if provided
	if repository != "" {
		if err := validateRepositoryName(repository); err != nil {
			fmt.Println(err.Error())
			osExit(1)
		}
	}

	var containerName string
	if repository == "" {
		containerName = container
	} else {
		containerName = repository
	}
	containerExists, err := docker.ContainerExists(containerName)
	if err != nil {
		// docker ps failed outright (daemon down, CLI missing) - stop with
		// the cause instead of stumbling into confusing downstream errors.
		fmt.Println("Error checking for existing container '" + containerName + "': " + err.Error())
		osExit(1)
	}
	if containerExists {
		fmt.Println("Container '" + containerName + "' already exists, name must be unique")
		osExit(1)
	}

	image, tag := splitImageTag(container)

	imageInfo, err := docker.InspectImage(image + ":" + tag)
	if err != nil {
		if _, err := docker.Pull(image + ":" + tag); err != nil {
			fmt.Printf("Warning: Failed to pull image %s:%s: %v\n", image, tag, err)
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

	fmt.Println("Creating repository " + containerName)
	repo := client.Repository{
		Name:       containerName,
		Properties: nil,
	}
	if createRepo {
		_, _, err := repositoriesApi.CreateRepository(ctx).Repository(repo).Execute()
		if err != nil && err.Error() == "409 Conflict" {
			fmt.Println("repository '" + repo.Name + "' already exists")
			osExit(1)
		}
	}

	argList := []string{"-d", "--label", "dev.dit.dit"}
	volArgs, metaVols, err := createRepoVolumes(docker, containerName, vols)
	if err != nil {
		return "", err
	}
	argList = append(argList, volArgs...)
	argList = append(argList, filterRunArgs(args, image+":"+tag)...)
	argList = append(argList, flagName)
	argList = append(argList, containerName)

	portArgs, metaPorts := portRunArgs(docker.GetSliceFromImage(image+":"+tag, "Config", "ExposedPorts"), disablePortMap)
	argList = append(argList, portArgs...)

	for _, env := range envVars {
		argList = append(argList, "--env")
		argList = append(argList, env)
	}

	if privileged {
		argList = append(argList, "--privileged")
	}

	repoDigest := firstRepoDigest(docker.GetValFromImage(image+":"+tag, "RepoDigests"))

	var dockerRunCmd string
	if len(repoDigest) == 0 {
		dockerRunCmd = image + ":" + tag
	} else {
		dockerRunCmd = repoDigest
	}

	metadata := map[string]interface{}{
		"v2": map[string]interface{}{
			"image": map[string]interface{}{
				"image":  image,
				"tag":    tag,
				"digest": repoDigest,
			},
			"environment":    envVars,
			"ports":          metaPorts,
			"volumes":        metaVols,
			"privileged":     privileged,
			"disablePortMap": disablePortMap,
		},
	}

	updateRepo := client.Repository{
		Name:       containerName,
		Properties: metadata,
	}
	_, _, err = repositoriesApi.UpdateRepository(ctx, containerName).Repository(updateRepo).Execute()
	if err != nil {
		fmt.Println(err)
		osExit(1)
	}
	_, err = docker.Run(dockerRunCmd, "", argList)

	/**
	The output from Run is used by the CLI and Clone, so the status and message need to ba passed up and handled.
	*/
	var m string
	if err != nil {
		m = err.Error()
	} else {
		m = "Running controlled container " + containerName
	}
	return m, err
}
