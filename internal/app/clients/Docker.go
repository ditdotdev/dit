package clients

import (
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/ditdotdev/dit/internal/app"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const EOL = "\n"

// defaultDockerHubRegistry is the upstream Docker Hub namespace where the
// official dit images live. Used as the fallback when no registry
// is configured and as the literal "dit" image-name lookup target.
const defaultDockerHubRegistry = "ditdotdev"

// dockerCmd is the name of the docker CLI binary that this client shells
// out to. Extracted so goconst doesn't flag the literal across the dozen
// ce.Exec(dockerCmd, ...) call sites below.
const dockerCmd = "docker"

// localRegistry is the sentinel registry value that means "use the image
// name as-is without prepending a registry namespace". Used by callers
// that hand-roll their own image references (e.g. `myrepo/myimage:tag`).
const localRegistry = "local"

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
		i = dockerCmd
	}
	if p == 0 {
		p = 5001
	}
	return docker{i, p, defaultDockerHubRegistry}
}

func DockerWithRegistry(i string, p int, r string) docker {
	if i == "" {
		i = dockerCmd
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
	if d.registry == localRegistry || strings.Contains(image, "/") {
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
		"--name=dit-" + d.identity + "-launch",
		"-v", "/var/lib:/var/lib",
		"-v", "/run/docker:/run/docker",
		"-v", "/lib:/var/lib/dit-" + d.identity + "/system",
		"-v", "dit-" + d.identity + "-data:/var/lib/dit-" + d.identity + "/data",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
	}
}

func (d docker) Version() (string, error) {
	return ce.Exec(dockerCmd, "-v")
}

func (d docker) ContainerExists(container string) (bool, error) {
	out, err := ce.Exec(dockerCmd, "ps", "-a", "-f", "name=^/"+container+`$`, "--format", `"{{.Names}}"`)
	return len(out) > 0, err
}

func (d docker) Pull(image string) (string, error) {
	return ce.Exec(dockerCmd, "pull", image)
}

func (d docker) Tag(source string, target string) (string, error) {
	return ce.Exec(dockerCmd, "tag", source, target)
}

func (d docker) Remove(container string, force bool) (string, error) {
	var args []string
	args = append(args, "rm")
	if force {
		args = append(args, "-f")
	}
	args = append(args, container)
	return ce.Exec(dockerCmd, args...)
}

func (d docker) RemoveStopped(repo string) (string, error) {
	c, _ := ce.Exec(dockerCmd, "ps", "-a", "-f", "name=^/"+repo+`$`, "--format", `"{{.ID}}"`)
	c = strings.ReplaceAll(c, EOL, "")
	c = strings.ReplaceAll(c, `"`, "")
	return ce.Exec(dockerCmd, "container", "rm", c)
}

func (d docker) RemoveVolume(name string, force bool) (string, error) {
	args := []string{
		"volume", "rm",
	}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)
	return ce.Exec(dockerCmd, args...)
}

// VolumeExists returns true if a Docker volume with the given name exists.
// `docker volume rm -f` exits 0 regardless of whether the volume was actually
// there, so callers that need to know whether removal was a no-op must check
// first with this helper.
func (d docker) VolumeExists(name string) bool {
	_, err := ce.Exec(dockerCmd, "volume", "inspect", name)
	return err == nil
}

func (d docker) InspectContainer(container string) (string, error) {
	return ce.Exec(dockerCmd, "inspect", "--type", "container", container)
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
	return ce.Exec(dockerCmd, "inspect", "--type", "image", image)
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
	return ce.Exec(dockerCmd, args...)
}

func (d docker) FetchLogs(container string) []string {
	output, _ := ce.Exec(dockerCmd, "logs", container)
	lines := strings.Split(output, EOL)
	return lines
}

func (d docker) DitLatestIsDownloaded(registry string, latest app.Version) bool {
	// If registry is the local sentinel, check for local dit:latest first
	if registry == localRegistry {
		localOut, _ := ce.Exec(dockerCmd, "images", defaultDockerHubRegistry, "--format", `"{{.Repository}}:{{.Tag}}"`)
		if strings.Contains(localOut, "dit:latest") {
			return true // Use local dit:latest image
		}
		// If no local image found, fall back to checking dit registry
		registry = defaultDockerHubRegistry
	}

	image := registry + "/dit"
	out, _ := ce.Exec(dockerCmd, "images", image, "--format", `"{{.Tag}}"`)
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
	out, err := ce.Exec(dockerCmd, "ps", "-f", "name=^/"+container+`$`, "--format", `"{{.Names}}"`)
	return len(out) > 0, err
}

func (d docker) DitServerIsAvailable() (bool, error) {
	return d.ContainerIsRunning("dit-" + d.identity + "-server")
}

func (d docker) DitLaunchIsAvailable() (bool, error) {
	return d.ContainerIsRunning("dit-" + d.identity + "-launch")
}

func (d docker) LaunchDitServers() (string, error) {
	ditImage := d.getImageName("dit:latest")
	args := d.getLocalLaunchArgs()
	args = append(
		args,
		"-e",
		"DIT_PORT="+strconv.Itoa(d.port),
		"-e",
		"DIT_IMAGE="+ditImage,
		"-e",
		"DIT_IDENTITY=dit-"+d.identity,
	)
	return d.Run(ditImage, "/bin/bash /dit/launch", args)
}

func (d docker) getKubernetesLaunchArgs() ([]string, error) {
	home, _ := os.UserHomeDir()
	srcKubeconfig := home + "/.kube/config"
	flatKubeconfig := home + "/.dit/kubeconfig-" + d.identity
	if err := FlattenKubeconfigToFile(srcKubeconfig, flatKubeconfig); err != nil {
		return nil, fmt.Errorf("preparing kubeconfig for server container: %w", err)
	}
	return []string{
		"-d",
		"--restart", "always",
		"--name=dit-" + d.identity + "-server",
		"-v", flatKubeconfig + ":/root/.kube/config:ro",
		"-v", "dit-" + d.identity + "-data:/var/lib/" + d.identity,
		"-e", "DIT_CONTEXT=kubernetes-csi",
		"-e", "DIT_IDENTITY=dit-" + d.identity,
		"-p", strconv.Itoa(d.port) + ":5001",
	}, nil
}

func (d docker) LaunchDitKubernetesServers() (string, error) {
	ditImage := d.getImageName("dit:latest")
	args, err := d.getKubernetesLaunchArgs()
	if err != nil {
		return "", err
	}
	// Forward DIT_* env vars from the CLI process to the server
	// container so operators can plumb server-side configuration (e.g.
	// DIT_K8S_POD_HOST_ALIASES for clusters where the remote
	// hostname isn't resolvable from inside pods) without needing a
	// dedicated CLI flag for every knob. Skip server-internal vars that
	// the launch args already set explicitly to avoid clobbering them.
	skip := map[string]bool{
		"DIT_PORT":     true,
		"DIT_IMAGE":    true,
		"DIT_IDENTITY": true,
		"DIT_CONTEXT":  true,
	}
	for _, kv := range os.Environ() {
		eq := strings.Index(kv, "=")
		if eq <= 0 {
			continue
		}
		k := kv[:eq]
		if !strings.HasPrefix(k, "DIT_") || skip[k] {
			continue
		}
		args = append(args, "-e", kv)
	}
	return d.Run(ditImage, "/bin/bash /dit/run", args)
}

func (d docker) FetchLaunchLogs() []string {
	return d.FetchLogs("dit-" + d.identity + "-launch")
}

func (d docker) TeardownDitServers() (string, error) {
	ditImage := d.getImageName("dit:latest")
	args := d.getLocalLaunchArgs()
	args = RemoveFromSlice(args, "-d")
	args = RemoveFromSlice(args, "--restart")
	args = RemoveFromSlice(args, "always")
	args = RemoveFromSlice(args, "--name=dit-"+d.identity+"-launch")
	args = append(args, "-e", "DIT_IDENTITY=dit-"+d.identity, "--rm")
	return d.Run(ditImage, "/bin/bash /dit/teardown", args)
}

func (d docker) RemoveDitImages(version string) (string, error) {
	var imageId, _ = ce.Exec(dockerCmd, "images", "dit:"+version, "--format", "{{.ID}}")
	imageId = strings.TrimSuffix(imageId, "\n")
	return ce.Exec(dockerCmd, "rmi", imageId, "-f")
}

func (d docker) RemoveDitServer() (string, error) {
	return d.Remove("dit-"+d.identity+"-server", true)
}

func (d docker) RemoveDitLaunch() (string, error) {
	return d.Remove("dit-"+d.identity+"-launch", true)
}

func (d docker) RemoveDitVolume() (string, error) {
	return d.RemoveVolume("dit-"+d.identity+"-data", false)
}

func (d docker) CreateVolume(name string, path string) (string, error) {
	return ce.Exec(dockerCmd, "volume", "create", "-d", "dit-"+d.identity, "-o", "path="+path, name)
}

func (d docker) ListVolumes(repo string) []string {
	var args []string
	var r []string
	args = append(args,
		"volume", "ls", "-f", "driver=dit-docker", "-f", "name="+repo,
		"--format", "{{.Name}}",
	)
	s, err := ce.Exec(dockerCmd, args...)
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
	return ce.Exec(dockerCmd, "stop", repo)
}

func (d docker) Start(repo string) (string, error) {
	return ce.Exec(dockerCmd, "start", repo)
}

func (d docker) Cp(source string, target string) (string, error) {
	// On Windows, convert MSYS2/Git Bash paths (e.g. /c/dev/...) to Windows paths (C:/dev/...)
	// so Docker Desktop can find the source files.
	if runtime.GOOS == "windows" && len(source) >= 3 && source[0] == '/' && source[2] == '/' {
		source = strings.ToUpper(string(source[1])) + ":" + source[2:]
	}
	return ce.Exec(dockerCmd, "cp", "-a", source+"/.", "dit-"+d.identity+"-server:"+target)
}
