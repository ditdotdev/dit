// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"fmt"
	"github.com/ditdotdev/dit/internal/app/providers/common"
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

// defaultTag is the tag assumed when a container reference has none.
const defaultTag = "latest"

// parseImageRef splits a container reference into image, tag and digest.
// The tag defaults to "latest" when the reference has none. The digest is
// the "sha256:…" part of an image:tag@sha256:… reference, which dit clone
// passes so the container starts from the exact committed image (#240).
func parseImageRef(ref string) (image, tag, digest string) {
	if at := strings.Index(ref, "@"); at >= 0 {
		digest = ref[at+1:]
		ref = ref[:at]
	}
	if !strings.Contains(ref, ":") {
		return ref, defaultTag, digest
	}
	parts := strings.Split(ref, ":")
	return parts[0], parts[1], digest
}

// imageLookupRef is the reference used for every docker inspect and run:
// image:tag, or image:tag@digest when a digest is known. Docker resolves
// the latter by the digest alone, so a local image:tag that points at a
// different image is never consulted.
func imageLookupRef(image, tag, digest string) string {
	ref := image + ":" + tag
	if digest != "" {
		ref += "@" + digest
	}
	return ref
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

func Run(container string, repository string, envVars []string, args []string, disablePortMap bool, privileged bool, pull common.PullPolicy, createRepo bool, port int, context string) (string, error) {
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

	image, tag, digest := parseImageRef(container)
	ref := imageLookupRef(image, tag, digest)

	if _, err := common.EnsureImage(docker, ref, "", pull); err != nil {
		fmt.Println(err)
		osExit(1)
	}
	imageInfo, _ := docker.InspectImage(ref)
	if len(imageInfo) == 0 {
		fmt.Println("Image information is not available")
		osExit(1)
	}
	vols := docker.GetSliceFromImage(ref, "Config", "Volumes")
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
	argList = append(argList, filterRunArgs(args, ref)...)
	argList = append(argList, flagName)
	argList = append(argList, containerName)

	portArgs, metaPorts := portRunArgs(docker.GetSliceFromImage(ref, "Config", "ExposedPorts"), disablePortMap)
	argList = append(argList, portArgs...)

	for _, env := range envVars {
		argList = append(argList, "--env")
		argList = append(argList, env)
	}

	if privileged {
		argList = append(argList, "--privileged")
	}

	repoDigest := ""
	if digests := common.RepoDigests(docker.GetValFromImage(ref, "RepoDigests")); len(digests) > 0 {
		repoDigest = digests[0]
	}

	var dockerRunCmd string
	if len(repoDigest) == 0 {
		dockerRunCmd = ref
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
