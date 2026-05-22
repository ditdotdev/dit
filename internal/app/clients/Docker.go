package clients

import (
	"datadatdat/internal/app"
	"fmt"
	"github.com/buger/jsonparser"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const EOL = "\n"

// defaultDockerHubRegistry is the upstream Docker Hub namespace where the
// official datadatdat images live. Used as the fallback when no registry
// is configured and as the literal "datadatdat" image-name lookup target.
const defaultDockerHubRegistry = "datadatdat"

type docker struct {
	identity string
	port     int
	registry string
}

// FormatVolumeName creates a Docker-compatible volume name using underscores
// Uses underscores for universal compatibility across all platforms
func (d docker) FormatVolumeName(repoName, volumeName string) string {
	return repoName + "_" + volumeName
}

func Docker(i string, p int) docker {
	if i == "" {
		i = "docker"
	}
	if p == 0 {
		p = 5001
	}
	return docker{i, p, defaultDockerHubRegistry}
}

func DockerWithRegistry(i string, p int, r string) docker {
	if i == "" {
		i = "docker"
	}
	if p == 0 {
		p = 5001
	}
	if r == "" {
		r = defaultDockerHubRegistry
	}
	return docker{i, p, r}
}

func (d docker) getImageName(image string) string {
	if d.registry == "local" || strings.Contains(image, "/") {
		return image
	}
	return d.registry + "/" + image
}

/*
*
https://yourbasic.org/golang/find-search-contains-slice/
*/
func Find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return len(a)
}

/*
*
https://stackoverflow.com/a/37335777
*/
func RemoveFromSlice(a []string, x string) []string {
	s := Find(a, x)
	return append(a[:s], a[s+1:]...)
}

func (d docker) GetIdentity() string {
	return d.identity
}

func (d docker) getLocalLaunchArgs() []string {
	return []string{
		"--privileged",
		"--pid=host",
		"--network=host",
		"-d",
		"--restart", "always",
		"--name=datadatdat-" + d.identity + "-launch",
		"-v", "/var/lib:/var/lib",
		"-v", "/run/docker:/run/docker",
		"-v", "/lib:/var/lib/datadatdat-" + d.identity + "/system",
		"-v", "datadatdat-" + d.identity + "-data:/var/lib/datadatdat-" + d.identity + "/data",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
	}
}

func (d docker) Version() (string, error) {
	return ce.Exec("docker", "-v")
}

func (d docker) ContainerExists(container string) (bool, error) {
	out, err := ce.Exec("docker", "ps", "-a", "-f", "name=^/"+container+`$`, "--format", `"{{.Names}}"`)
	return len(out) > 0, err
}

func (d docker) Pull(image string) (string, error) {
	return ce.Exec("docker", "pull", image)
}

func (d docker) Tag(source string, target string) (string, error) {
	return ce.Exec("docker", "tag", source, target)
}

func (d docker) Remove(container string, force bool) (string, error) {
	var args []string
	args = append(args, "rm")
	if force {
		args = append(args, "-f")
	}
	args = append(args, container)
	return ce.Exec("docker", args...)
}

func (d docker) RemoveStopped(repo string) (string, error) {
	c, _ := ce.Exec("docker", "ps", "-a", "-f", "name=^/"+repo+`$`, "--format", `"{{.ID}}"`)
	c = strings.ReplaceAll(c, EOL, "")
	c = strings.ReplaceAll(c, `"`, "")
	return ce.Exec("docker", "container", "rm", c)
}

func (d docker) RemoveVolume(name string, force bool) (string, error) {
	args := []string{
		"volume", "rm",
	}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)
	return ce.Exec("docker", args...)
}

// VolumeExists returns true if a Docker volume with the given name exists.
// `docker volume rm -f` exits 0 regardless of whether the volume was actually
// there, so callers that need to know whether removal was a no-op must check
// first with this helper.
func (d docker) VolumeExists(name string) bool {
	_, err := ce.Exec("docker", "volume", "inspect", name)
	return err == nil
}

func (d docker) InspectContainer(container string) (string, error) {
	return ce.Exec("docker", "inspect", "--type", "container", container)
}

func (d docker) GetValFromContainer(c string, key ...string) (string, error) {
	key = append([]string{"[0]"}, key...)
	result, err := d.InspectContainer(c)
	out, _, _, _ := jsonparser.Get([]byte(result), key...)
	return string(out), err
}

func (d docker) GetSliceFromContainer(c string, key ...string) []string {
	raw, _ := d.GetValFromContainer(c, key...)
	raw = strings.TrimLeft(raw, "[")
	raw = strings.TrimLeft(raw, "{")
	raw = strings.TrimRight(raw, "}")
	raw = strings.TrimRight(raw, "]")
	raw = strings.ReplaceAll(raw, " ", "") //TODO trimspace
	raw = strings.ReplaceAll(raw, EOL, "")
	out := strings.Split(raw, ",")
	return out
}

func (d docker) InspectImage(image string) (string, error) {
	return ce.Exec("docker", "inspect", "--type", "image", image)
}

func (d docker) GetValFromImage(image string, key ...string) string {
	key = append([]string{"[0]"}, key...)
	result, _ := d.InspectImage(image)
	out, _, _, _ := jsonparser.Get([]byte(result), key...)
	return string(out)
}

func (d docker) GetSliceFromImage(image string, key ...string) []string {
	raw := d.GetValFromImage(image, key...)
	raw = strings.TrimLeft(raw, "{")
	raw = strings.TrimRight(raw, "}")
	raw = strings.ReplaceAll(raw, " ", "") //TODO trimspace
	raw = strings.ReplaceAll(raw, EOL, "")
	out := strings.Split(raw, ",")
	return out
}

func (d docker) Run(image string, entry string, args []string) (string, error) {
	args = append([]string{"run"}, args...)
	args = append(args, image)
	if len(entry) > 0 {
		args = append(args, strings.Split(entry, " ")...)
	}
	return ce.Exec("docker", args...)
}

func (d docker) FetchLogs(container string) []string {
	output, _ := ce.Exec("docker", "logs", container)
	lines := strings.Split(output, EOL)
	return lines
}

func (d docker) DatadatdatLatestIsDownloaded(registry string, latest app.Version) bool {
	// If registry is "local", check for local datadatdat:latest first
	if registry == "local" {
		localOut, _ := ce.Exec("docker", "images", defaultDockerHubRegistry, "--format", `"{{.Repository}}:{{.Tag}}"`)
		if strings.Contains(localOut, "datadatdat:latest") {
			return true // Use local datadatdat:latest image
		}
		// If no local image found, fall back to checking datadatdat registry
		registry = defaultDockerHubRegistry
	}

	image := registry + "/datadatdat"
	out, _ := ce.Exec("docker", "images", image, "--format", `"{{.Tag}}"`)
	tags := strings.Split(string(out), EOL)
	hasVersionTag := false
	for _, item := range tags {
		tag := strings.Trim(item, "\"")
		if tag != "latest" && tag != "" {
			v := app.Version{}.FromString(tag)
			if v.Compare(latest) == 0 {
				hasVersionTag = true
				break
			}
		}
	}

	if !hasVersionTag {
		return false
	}

	// Image exists locally. Check if it was pulled from a registry (has
	// RepoDigests) or built locally (no RepoDigests). Locally-built images
	// are trusted as-is; registry-pulled images may be stale and should be
	// re-pulled to pick up newer builds under the same tag.
	repoDigests := d.GetValFromImage(image+":latest", "RepoDigests")
	if repoDigests == "" || repoDigests == "null" {
		// No RepoDigests means locally-built image -- trust it
		return true
	}
	// Registry-pulled image -- treat as stale so the caller re-pulls
	return false
}

func (d docker) ContainerIsRunning(container string) (bool, error) {
	out, err := ce.Exec("docker", "ps", "-f", "name=^/"+container+`$`, "--format", `"{{.Names}}"`)
	return len(out) > 0, err
}

func (d docker) DatadatdatServerIsAvailable() (bool, error) {
	return d.ContainerIsRunning("datadatdat-" + d.identity + "-server")
}

func (d docker) DatadatdatLaunchIsAvailable() (bool, error) {
	return d.ContainerIsRunning("datadatdat-" + d.identity + "-launch")
}

func (d docker) LaunchDatadatdatServers() (string, error) {
	datadatdatImage := d.getImageName("datadatdat:latest")
	args := d.getLocalLaunchArgs()
	args = append(
		args,
		"-e",
		"DATADATDAT_PORT="+strconv.Itoa(d.port),
		"-e",
		"DATADATDAT_IMAGE="+datadatdatImage,
		"-e",
		"DATADATDAT_IDENTITY=datadatdat-"+d.identity,
	)
	return d.Run(datadatdatImage, "/bin/bash /datadatdat/launch", args)
}

func (d docker) getKubernetesLaunchArgs() ([]string, error) {
	home, _ := os.UserHomeDir()
	srcKubeconfig := home + "/.kube/config"
	flatKubeconfig := home + "/.datadatdat/kubeconfig-" + d.identity
	if err := FlattenKubeconfigToFile(srcKubeconfig, flatKubeconfig); err != nil {
		return nil, fmt.Errorf("preparing kubeconfig for server container: %w", err)
	}
	return []string{
		"-d",
		"--restart", "always",
		"--name=datadatdat-" + d.identity + "-server",
		"-v", flatKubeconfig + ":/root/.kube/config:ro",
		"-v", "datadatdat-" + d.identity + "-data:/var/lib/" + d.identity,
		"-e", "DATADATDAT_CONTEXT=kubernetes-csi",
		"-e", "DATADATDAT_IDENTITY=datadatdat-" + d.identity,
		"-p", strconv.Itoa(d.port) + ":5001",
	}, nil
}

func (d docker) LaunchDatadatdatKubernetesServers() (string, error) {
	datadatdatImage := d.getImageName("datadatdat:latest")
	args, err := d.getKubernetesLaunchArgs()
	if err != nil {
		return "", err
	}
	// Forward DATADATDAT_* env vars from the CLI process to the server
	// container so operators can plumb server-side configuration (e.g.
	// DATADATDAT_K8S_POD_HOST_ALIASES for clusters where the remote
	// hostname isn't resolvable from inside pods) without needing a
	// dedicated CLI flag for every knob. Skip server-internal vars that
	// the launch args already set explicitly to avoid clobbering them.
	skip := map[string]bool{
		"DATADATDAT_PORT":     true,
		"DATADATDAT_IMAGE":    true,
		"DATADATDAT_IDENTITY": true,
		"DATADATDAT_CONTEXT":  true,
	}
	for _, kv := range os.Environ() {
		eq := strings.Index(kv, "=")
		if eq <= 0 {
			continue
		}
		k := kv[:eq]
		if !strings.HasPrefix(k, "DATADATDAT_") || skip[k] {
			continue
		}
		args = append(args, "-e", kv)
	}
	return d.Run(datadatdatImage, "/bin/bash /datadatdat/run", args)
}

func (d docker) FetchLaunchLogs() []string {
	return d.FetchLogs("datadatdat-" + d.identity + "-launch")
}

func (d docker) TeardownDatadatdatServers() (string, error) {
	datadatdatImage := d.getImageName("datadatdat:latest")
	args := d.getLocalLaunchArgs()
	args = RemoveFromSlice(args, "-d")
	args = RemoveFromSlice(args, "--restart")
	args = RemoveFromSlice(args, "always")
	args = RemoveFromSlice(args, "--name=datadatdat-"+d.identity+"-launch")
	args = append(args, "-e", "DATADATDAT_IDENTITY=datadatdat-"+d.identity, "--rm")
	return d.Run(datadatdatImage, "/bin/bash /datadatdat/teardown", args)
}

func (d docker) RemoveDatadatdatImages(version string) (string, error) {
	var imageId, _ = ce.Exec("docker", "images", "datadatdat:"+version, "--format", "{{.ID}}")
	imageId = strings.TrimSuffix(imageId, "\n")
	return ce.Exec("docker", "rmi", imageId, "-f")
}

func (d docker) RemoveDatadatdatServer() (string, error) {
	return d.Remove("datadatdat-"+d.identity+"-server", true)
}

func (d docker) RemoveDatadatdatLaunch() (string, error) {
	return d.Remove("datadatdat-"+d.identity+"-launch", true)
}

func (d docker) RemoveDatadatdatVolume() (string, error) {
	return d.RemoveVolume("datadatdat-"+d.identity+"-data", false)
}

func (d docker) CreateVolume(name string, path string) (string, error) {
	return ce.Exec("docker", "volume", "create", "-d", "datadatdat-"+d.identity, "-o", "path="+path, name)
}

func (d docker) ListVolumes(repo string) []string {
	var args []string
	var r []string
	args = append(args,
		"volume", "ls", "-f", "driver=datadatdat-docker", "-f", "name="+repo,
		"--format", "{{.Name}}",
	)
	s, err := ce.Exec("docker", args...)
	if err == nil {
		vols := strings.Split(s, "\n")
		vols = vols[:len(vols)-1]
		for _, v := range vols {
			if strings.Contains(v, repo+"_v") {
				r = append(r, v)
			}
		}
	}
	return r
}

func (d docker) Stop(repo string) (string, error) {
	return ce.Exec("docker", "stop", repo)
}

func (d docker) Start(repo string) (string, error) {
	return ce.Exec("docker", "start", repo)
}

func (d docker) Cp(source string, target string) (string, error) {
	// On Windows, convert MSYS2/Git Bash paths (e.g. /c/dev/...) to Windows paths (C:/dev/...)
	// so Docker Desktop can find the source files.
	if runtime.GOOS == "windows" && len(source) >= 3 && source[0] == '/' && source[2] == '/' {
		source = strings.ToUpper(string(source[1])) + ":" + source[2:]
	}
	return ce.Exec("docker", "cp", "-a", source+"/.", "datadatdat-"+d.identity+"-server:"+target)
}
