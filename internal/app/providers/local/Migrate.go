package local

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	datadatdatclient "github.com/datadatdat/datadatdat-client-go"
)

func getLocalSrcFromPath(path string, mounts []mount) string {
	var r string
	for _, m := range mounts {
		if m.Destination == path {
			r = m.Source
		}
	}
	return r
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
	args = append(args, "-d", "--label", "com.datadatdat.datadatdat")
	repo := datadatdatclient.Repository{
		Name:       name,
		Properties: make(map[string]interface{}),
	}
	if _, _, err := repositoriesApi.CreateRepository(ctx).Repository(repo).Execute(); err != nil {
		fmt.Printf("Error creating repository: %v\n", err)
		return
	}
	m, _ := docker.GetValFromContainer(container, "Mounts")
	var mounts []mount
	if err = json.Unmarshal([]byte(m), &mounts); err != nil {
		fmt.Printf("Error unmarshaling mounts: %v\n", err)
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
		localSrc := getLocalSrcFromPath(path, mounts)
		if localSrc != "" {
			fmt.Println("Copying data to " + volName)
			if _, err := volumesApi.ActivateVolume(ctx, name, v).Execute(); err != nil {
				fmt.Printf("Warning: Failed to activate volume %s: %v\n", v, err)
				continue
			}
			vol, _, _ := volumesApi.GetVolume(ctx, name, v).Execute()
			target := fmt.Sprintf("%v", vol.Config["mountpoint"])
			if _, err := docker.Cp(localSrc, target); err != nil {
				fmt.Printf("Warning: Failed to copy data to volume: %v\n", err)
			}
			if _, err := volumesApi.DeactivateVolume(ctx, name, v).Execute(); err != nil {
				fmt.Printf("Warning: Failed to deactivate volume %s: %v\n", v, err)
			}
		}
		args = append(args, "--mount", "type=volume,src="+volName+",dst="+path+",volume-driver=datadatdat-"+docker.GetIdentity())
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

	updateRepo := datadatdatclient.Repository{
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
