// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	ditclient "github.com/ditdotdev/dit-client-go"
	"github.com/ditdotdev/dit/internal/app"
	"github.com/ditdotdev/dit/internal/app/clients"
	"os"
)

// dockerClient is the subset of clients.Docker methods used by providers/kubernetes.
// Exists so package functions can be unit-tested with a mock instead of a real
// docker daemon.
type dockerClient interface {
	DitLatestIsDownloaded(registry string, latest app.Version) bool
	DitLaunchIsAvailable() (bool, error)
	DitServerIsAvailable() (bool, error)
	GetSliceFromImage(image string, key ...string) []string
	GetValFromContainer(c string, key ...string) (string, error)
	GetValFromImage(image string, key ...string) string
	InspectImage(image string) (string, error)
	LaunchDitKubernetesServers() (string, error)
	Pull(image string) (string, error)
	Remove(container string, force bool) (string, error)
	RemoveDitImages(version string) (string, error)
	RemoveVolume(name string, force bool) (string, error)
	Tag(source string, target string) (string, error)
	Version() (string, error)
	VolumeExists(name string) bool
}

// kubernetesClient is the subset of clients.Kubernetes methods used by
// providers/kubernetes. Exists so package functions can be unit-tested with a
// mock kubernetes client.
type kubernetesClient interface {
	CreateStatefulSet(repoName string, imageId string, ports []int, volumes []ditclient.Volume, environment []string) error
	GetStatefulSetStatus(repoName string) (string, error)
	WaitForStatefulSet(repoName string)
	StartPortForwarding(repoName string)
	StopPortForwarding(repoName string)
	UpdateStatefulSetVolumes(repoName string, volumes []ditclient.Volume)
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

// SetOsExitForTesting swaps the package-level osExit seam, returning the
// previous value so callers can restore it on cleanup. Lets tests in the
// parent providers package intercept the exit calls kubernetes.* helpers
// make on unavailable backing services.
func SetOsExitForTesting(fn func(int)) func(int) {
	prev := osExit
	osExit = fn
	return prev
}

// UseNoopK8sForTesting swaps the package-level k8s client for a no-op
// implementation that returns zero values and never blocks. Test callers
// in the parent providers package use this to keep the long-poll
// WaitForStatefulSet path from hanging the test suite. Returns a restore
// function that callers should defer.
func UseNoopK8sForTesting() func() {
	prev := k8s
	k8s = noopK8s{}
	return func() { k8s = prev }
}

// noopK8sRunningStatus is the canned response noopK8s returns from
// GetStatefulSetStatus. Extracted to keep goconst quiet about the
// "running" literal showing up in multiple files in the package.
const noopK8sRunningStatus = statusRunning

type noopK8s struct{}

func (noopK8s) CreateStatefulSet(string, string, []int, []ditclient.Volume, []string) error {
	return nil
}
func (noopK8s) GetStatefulSetStatus(string) (string, error) {
	return noopK8sRunningStatus, nil
}
func (noopK8s) WaitForStatefulSet(string)                           {}
func (noopK8s) StartPortForwarding(string)                          {}
func (noopK8s) StopPortForwarding(string)                           {}
func (noopK8s) UpdateStatefulSetVolumes(string, []ditclient.Volume) {}
func (noopK8s) DeleteStatefulSpec(string)                           {}
func (noopK8s) StopStatefulSet(string)                              {}
func (noopK8s) StartStatefulSet(string)                             {}
