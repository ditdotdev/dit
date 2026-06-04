package local

import (
	"github.com/ditdotdev/dit/internal/app"
	"github.com/ditdotdev/dit/internal/app/clients"
	"os"
)

// dockerClient is the subset of clients.Docker methods used by providers/local.
// It exists as a local interface so package functions can be unit-tested with a
// mock implementation instead of a real docker daemon. The concrete type
// returned by clients.Docker/DockerWithRegistry satisfies it structurally.
type dockerClient interface {
	ContainerExists(container string) (bool, error)
	ContainerIsRunning(container string) (bool, error)
	Cp(source string, target string) (string, error)
	CreateVolume(name string, path string) (string, error)
	DitLatestIsDownloaded(registry string, latest app.Version) bool
	DitLaunchIsAvailable() (bool, error)
	DitServerIsAvailable() (bool, error)
	FetchLaunchLogs() []string
	FormatVolumeName(repoName, volumeName string) string
	GetIdentity() string
	GetSliceFromContainer(c string, key ...string) []string
	GetSliceFromImage(image string, key ...string) []string
	GetValFromContainer(c string, key ...string) (string, error)
	GetValFromImage(image string, key ...string) string
	InspectContainer(container string) (string, error)
	InspectImage(image string) (string, error)
	LaunchDitServers() (string, error)
	ListVolumes(repo string) []string
	Pull(image string) (string, error)
	Remove(container string, force bool) (string, error)
	RemoveDitImages(version string) (string, error)
	RemoveDitLaunch() (string, error)
	RemoveDitServer() (string, error)
	RemoveDitVolume() (string, error)
	RemoveStopped(repo string) (string, error)
	RemoveVolume(name string, force bool) (string, error)
	Run(image string, entry string, args []string) (string, error)
	Start(repo string) (string, error)
	Stop(repo string) (string, error)
	Tag(source string, target string) (string, error)
	TeardownDitServers() (string, error)
	Version() (string, error)
	VolumeExists(name string) bool
}

// newDocker and newDockerWithRegistry are var-bound factories so tests can
// substitute a mock dockerClient without going through clients.Docker.
var newDocker = func(context string, port int) dockerClient {
	return clients.Docker(context, port)
}

var newDockerWithRegistry = func(context string, port int, registry string) dockerClient {
	return clients.DockerWithRegistry(context, port, registry)
}

// osExit indirects os.Exit so tests can capture the requested exit code.
var osExit = os.Exit

// SetOsExitForTesting swaps the package-level osExit seam, returning the
// previous value so callers can restore it on cleanup. Lets tests in
// the parent providers package intercept the exit calls that local.*
// helpers make when docker isn't available.
func SetOsExitForTesting(fn func(int)) func(int) {
	prev := osExit
	osExit = fn
	return prev
}
