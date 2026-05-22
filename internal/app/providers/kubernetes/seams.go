package kubernetes

import (
	"datadatdat/internal/app"
	"datadatdat/internal/app/clients"
	datadatdatclient "github.com/datadatdat/datadatdat-client-go"
	"os"
)

// dockerClient is the subset of clients.Docker methods used by providers/kubernetes.
// Exists so package functions can be unit-tested with a mock instead of a real
// docker daemon.
type dockerClient interface {
	DatadatdatLatestIsDownloaded(registry string, latest app.Version) bool
	DatadatdatLaunchIsAvailable() (bool, error)
	DatadatdatServerIsAvailable() (bool, error)
	FetchLaunchLogs() []string
	GetSliceFromImage(image string, key ...string) []string
	GetValFromContainer(c string, key ...string) (string, error)
	GetValFromImage(image string, key ...string) string
	InspectImage(image string) (string, error)
	LaunchDatadatdatKubernetesServers() (string, error)
	Pull(image string) (string, error)
	Remove(container string, force bool) (string, error)
	RemoveDatadatdatImages(version string) (string, error)
	RemoveVolume(name string, force bool) (string, error)
	Tag(source string, target string) (string, error)
	Version() (string, error)
	VolumeExists(name string) bool
}

// kubernetesClient is the subset of clients.Kubernetes methods used by
// providers/kubernetes. Exists so package functions can be unit-tested with a
// mock kubernetes client.
type kubernetesClient interface {
	CreateStatefulSet(repoName string, imageId string, ports []int, volumes []datadatdatclient.Volume, environment []string) error
	GetStatefulSetStatus(repoName string) (string, error)
	WaitForStatefulSet(repoName string)
	StartPortForwarding(repoName string)
	StopPortForwarding(repoName string)
	UpdateStatefulSetVolumes(repoName string, volumes []datadatdatclient.Volume)
	DeleteStatefulSpec(repoName string)
	StopStatefulSet(repoName string)
	StartStatefulSet(repoName string)
}

// newDocker is a var-bound factory so tests can substitute a mock dockerClient
// without going through clients.Docker. k8s is an interface-typed package var
// so tests can swap in a mock kubernetesClient.
var newDocker = func(context string, port int) dockerClient {
	return clients.Docker(context, port)
}

var k8s kubernetesClient = clients.Kubernetes("default", "localhost", 5001)

// osExit indirects os.Exit so tests can capture the requested exit code.
var osExit = os.Exit
