package local

import (
	"fmt"
	"strconv"
	"strings"

	client "github.com/ditdotdev/dit-client-go"
)

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
	containerExists, _ := docker.ContainerExists(containerName) //TODO handle this error
	if containerExists {
		fmt.Println("Container '" + containerName + "' already exists, name must be unique")
		osExit(1)
	}

	var image string
	if strings.Contains(container, ":") {
		image = strings.Split(container, ":")[0]
	} else {
		image = container
	}

	var tag string
	if strings.Contains(container, ":") {
		containerParts := strings.Split(container, ":")
		if len(containerParts) > 1 {
			tag = containerParts[1]
		} else {
			tag = "latest"
		}
	} else {
		tag = "latest"
	}

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
	var metaVols []map[string]string
	for i, path := range vols {
		volumeName := "v" + strconv.Itoa(i)
		volName := docker.FormatVolumeName(containerName, volumeName)
		path := strings.Split(path, ":")[0]
		path = strings.ReplaceAll(path, `"`, "")

		fmt.Println("Creating docker volume " + volName + " with path " + path)
		_, err := docker.CreateVolume(volName, path)
		if err != nil {
			return "", err
		}
		argList = append(argList, "-v")
		argList = append(argList, volName+":"+path)
		addVol := make(map[string]string)
		addVol["name"] = volumeName
		addVol["path"] = path
		metaVols = append(metaVols, addVol)
	}
	// Filter out --name and image arguments from args
	var filteredArgs []string
	imageTag := image + ":" + tag
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
			filteredArgs = append(filteredArgs, args[i])
			i += 1
		}
	}
	argList = append(argList, filteredArgs...)
	argList = append(argList, flagName)
	argList = append(argList, containerName)

	var metaPorts []map[string]string
	ports := docker.GetSliceFromImage(image+":"+tag, "Config", "ExposedPorts")
	for _, rawPort := range ports {
		rawPort = strings.ReplaceAll(rawPort, `"`, "")
		portParts := strings.Split(rawPort, "/")
		if len(portParts) < 2 {
			continue // Skip malformed port entries
		}
		port := portParts[0]
		protocolPart := portParts[1]
		protocol := strings.Split(protocolPart, ":")[0]
		if !disablePortMap {
			argList = append(argList, "-p")
			argList = append(argList, port+":"+port+"/"+protocol)
		}
		addPort := make(map[string]string)
		addPort["protocol"] = protocol
		addPort["port"] = port
		metaPorts = append(metaPorts, addPort)
	}

	for _, env := range envVars {
		argList = append(argList, "--env")
		argList = append(argList, env)
	}

	if privileged {
		argList = append(argList, "--privileged")
	}

	repoDigest := docker.GetValFromImage(image+":"+tag, "RepoDigests")
	repoDigest = strings.ReplaceAll(repoDigest, "[", "")
	repoDigest = strings.ReplaceAll(repoDigest, "]", "")
	repoDigest = strings.ReplaceAll(repoDigest, " ", "")
	repoDigest = strings.ReplaceAll(repoDigest, `"`, "")
	repoDigest = strings.ReplaceAll(repoDigest, "\n", "")
	repoDigest = strings.TrimSpace(repoDigest)

	// If multiple digests are present (separated by commas), take the first one
	if strings.Contains(repoDigest, ",") {
		digestParts := strings.Split(repoDigest, ",")
		repoDigest = strings.TrimSpace(digestParts[0])
	}

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
