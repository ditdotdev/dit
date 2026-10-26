// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	ditclient "github.com/ditdotdev/dit-client-go"
)

// copyVolumeFromContainer activates dit volume v of repository name, copies
// the source container's path onto its ZFS mountpoint, and deactivates it
// again. The deactivate is best effort on the failure path so the caller
// can roll back.
func copyVolumeFromContainer(docker dockerClient, container string, name string, v string, path string) error {
	if _, err := volumesApi.ActivateVolume(ctx, name, v).Execute(); err != nil {
		return fmt.Errorf("activate: %w", err)
	}
	vol, _, err := volumesApi.GetVolume(ctx, name, v).Execute()
	if err != nil {
		_, _ = volumesApi.DeactivateVolume(ctx, name, v).Execute()
		return fmt.Errorf("get volume: %w", err)
	}
	target := fmt.Sprintf("%v", vol.Config["mountpoint"])
	if _, err := docker.CpFromContainer(container, path, target); err != nil {
		_, _ = volumesApi.DeactivateVolume(ctx, name, v).Execute()
		return err
	}
	if _, err := volumesApi.DeactivateVolume(ctx, name, v).Execute(); err != nil {
		return fmt.Errorf("deactivate: %w", err)
	}
	return nil
}

// rollbackMigrate removes everything Migrate created before the data copy
// failed: the dit volumes and the repository record. The repository
// container has not been started at that point, and the source container
// is never touched.
func rollbackMigrate(container string, name string, port int, context string) {
	fmt.Println("Rolling back repository " + name)
	Remove(name, true, port, context)
	fmt.Println("Rolled back repository " + name + "; " + container + " is unchanged")
}

type Commit func(string, string, []string, string, string, int)

func Migrate(container string, name string, user string, email string, commit Commit, port int, context string) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := newDocker(context, port)

	_, err := docker.InspectContainer(container)
	if err != nil {
		fmt.Println("Container information is not available")
		osExit(1)
	}
	r, _ := docker.GetValFromContainer(container, "State", "Running")
	running, _ := strconv.ParseBool(r)
	if running {
		fmt.Println("Cannot migrate a running container. Please stop " + container)
		osExit(1)
	}

	// Validate repository name
	if err := validateRepositoryName(name); err != nil {
		fmt.Println(err.Error())
		osExit(1)
	}
	image, _ := docker.GetValFromContainer(container, "Image")
	_, err = docker.InspectImage(image)
	if err != nil {
		fmt.Println("Image information is not available")
		osExit(1)
	}
	vols := docker.GetSliceFromImage(image, "Config", "Volumes")
	if len(vols) == 0 {
		fmt.Println("No volumes found for image " + image)
		osExit(1)
	}
	fmt.Println("Creating repository " + name)
	var args []string
	args = append(args, "-d", "--label", "dev.dit.dit")
	repo := ditclient.Repository{
		Name:       name,
		Properties: make(map[string]interface{}),
	}
	if _, _, err := repositoriesApi.CreateRepository(ctx).Repository(repo).Execute(); err != nil {
		fmt.Printf("Error creating repository: %v\n", err)
		return
	}
	for i, p := range vols {
		path := strings.Split(p, ":")[0]
		path = strings.ReplaceAll(path, `"`, "")
		v := "v" + strconv.Itoa(i)
		volName := docker.FormatVolumeName(name, v)
		fmt.Println("Creating docker volume " + volName + " with path " + path)
		if _, err := docker.CreateVolume(volName, path); err != nil {
			fmt.Printf("Error creating volume %s: %v\n", volName, err)
			continue
		}
		// Copy whatever the source container holds at path, however Docker
		// backs it (anonymous volume, named volume, bind mount, or the
		// writable layer). The copy runs inside the dit server, so no host
		// path is involved (#256).
		fmt.Println("Copying data to " + volName)
		if err := copyVolumeFromContainer(docker, container, name, v, path); err != nil {
			fmt.Printf("Error: failed to copy data to volume %s: %v\n", v, err)
			rollbackMigrate(container, name, port, context)
			osExit(1)
			return
		}
		args = append(args, "--mount", "type=volume,src="+volName+",dst="+path+",volume-driver=dit-"+docker.GetIdentity())
	}

	e := docker.GetSliceFromContainer(container, "Config", "Env")
	for _, env := range e {
		args = append(args, "-e", strings.Trim(env, "\""))
	}

	p, _ := docker.GetValFromContainer(container, "HostConfig", "PortBindings")
	var ports map[string][]map[string]string
	if err := json.Unmarshal([]byte(p), &ports); err != nil {
		fmt.Printf("Warning: Failed to unmarshal port bindings: %v\n", err)
	} else {
		for k, port := range ports {
			containerPort := strings.Split(k, "/")[0]
			hostIp, ok := port[0]["HostIp"]
			hostPort := port[0]["HostPort"]
			args = append(args, "-p")
			if ok && hostIp != "" {
				args = append(args, hostIp+":"+hostPort+":"+containerPort)
			} else {
				args = append(args, hostPort+":"+containerPort)
			}
		}
	}
	args = append(args, "--name", name)
	repoDigest := docker.GetSliceFromImage(image, "RepoDigests")[0]
	repoDigest = strings.TrimLeft(repoDigest, `["`)
	repoDigest = strings.TrimRight(repoDigest, `"]`)

	metadata := make(map[string]interface{})
	metadata["container"] = repoDigest
	metadata["runtime"] = strings.Join(args, " ")

	updateRepo := ditclient.Repository{
		Name:       name,
		Properties: metadata,
	}
	if _, _, err := repositoriesApi.UpdateRepository(ctx, name).Repository(updateRepo).Execute(); err != nil {
		fmt.Printf("Warning: Failed to update repository metadata: %v\n", err)
	}
	_, err = docker.Run(image, "", args)
	if err != nil {
		fmt.Println(err)
		osExit(1)
	}
	commit(name, "Initial Migration", nil, user, email, port)
	fmt.Println(container + " migrated to controlled environment " + name)
}
