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

// migratePaths returns the container paths Migrate copies into the new
// repository, or a user-facing problem when there are none. The --volume
// override wins over the image's declared volumes; each override path must
// be absolute. With neither, the caller has nothing to tell dit where the
// data lives (#258).
func migratePaths(docker dockerClient, image string, override []string) ([]string, string) {
	if len(override) > 0 {
		for _, p := range override {
			if !strings.HasPrefix(p, "/") {
				return nil, fmt.Sprintf("--volume path %q must be absolute", p)
			}
		}
		if len(docker.GetSliceFromImage(image, "Config", "Volumes")) > 0 {
			fmt.Println("Using volume paths from --volume instead of the image's declared volumes: " + strings.Join(override, ", "))
		}
		return override, ""
	}
	var paths []string
	for _, p := range docker.GetSliceFromImage(image, "Config", "Volumes") {
		path := strings.Split(p, ":")[0]
		paths = append(paths, strings.ReplaceAll(path, `"`, ""))
	}
	if len(paths) == 0 {
		return nil, "No volumes found for image " + image + "; pass --volume <container-path> to name the data to copy"
	}
	return paths, ""
}

// Migrate creates repository name from the stopped container. volumes, when
// non-empty, lists the container paths to copy instead of the image's
// declared volumes.
func Migrate(container string, name string, volumes []string, user string, email string, commit Commit, port int, context string) {
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
	paths, problem := migratePaths(docker, image, volumes)
	if problem != "" {
		fmt.Println(problem)
		osExit(1)
		return
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
	for i, path := range paths {
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

	// A source that ran privileged (Db2 needs it to start) must stay
	// privileged, or the migrated engine never comes up. The flag is kept
	// in the runtime metadata too, so a clone of the repository gets it.
	if priv, _ := docker.GetValFromContainer(container, "HostConfig", "Privileged"); priv == "true" {
		args = append(args, "--privileged")
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
