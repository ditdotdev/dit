// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	ditclient "github.com/ditdotdev/dit-client-go"
	"github.com/ditdotdev/dit/internal/app"
	"github.com/ditdotdev/dit/internal/app/utils"

	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"
)

// init shrinks polling intervals + timeouts so tests don't sit on a second.
func init() {
	CommitReadyPollInterval = 1 * time.Millisecond
	CommitReadyTimeout = 100 * time.Millisecond
	utils.MonitorPollInterval = 1 * time.Millisecond
	utils.MonitorIdleTimeout = 100 * time.Millisecond
	serverReadyPollInterval = 1 * time.Millisecond
	serverReadyTimeout = 5 * time.Millisecond
	serverPing = func(string) bool { return true }
}

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func startMockServer(t *testing.T, handler http.HandlerFunc) int {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	p, _ := strconv.Atoi(u.Port())
	return p
}

type exitPanic struct{ code int }

func captureExit(t *testing.T, fn func()) (didExit bool, code int) {
	t.Helper()
	original := osExit
	defer func() { osExit = original }()
	osExit = func(c int) { panic(exitPanic{c}) }
	defer func() {
		if r := recover(); r != nil {
			if p, ok := r.(exitPanic); ok {
				didExit = true
				code = p.code
				return
			}
			panic(r)
		}
	}()
	fn()
	return false, 0
}

// fakeDocker satisfies the dockerClient interface in providers/kubernetes.
type fakeDocker struct {
	versionErr          error
	latestDownloaded    bool
	launchAvailable     bool
	serverAvailable     bool
	getSliceFromImage   map[string][]string
	getValFromContainer map[string]string
	getValFromImage     map[string]string
	inspectImageOut     string
	inspectImageErr     error
	launchK8sOut        string
	launchK8sErr        error
	pullErr             error
	removeErr           error
	removeImagesErr     error
	removeVolumeErr     error
	tagErr              error
	volumeExists        map[string]bool

	PullCalls   int
	RemoveCalls int
	// RemovedNames records the container names passed to Remove, so tests
	// can assert the cleanup targets context-derived names (#214).
	RemovedNames []string
}

func (f *fakeDocker) DitLatestIsDownloaded(string, app.Version) bool {
	return f.latestDownloaded
}
func (f *fakeDocker) DitLaunchIsAvailable() (bool, error) { return f.launchAvailable, nil }
func (f *fakeDocker) DitServerIsAvailable() (bool, error) { return f.serverAvailable, nil }
func (f *fakeDocker) GetSliceFromImage(i string, k ...string) []string {
	return f.getSliceFromImage[i+":"+joinKeys(k)]
}
func (f *fakeDocker) GetValFromContainer(c string, k ...string) (string, error) {
	return f.getValFromContainer[c+":"+joinKeys(k)], nil
}
func (f *fakeDocker) GetValFromImage(i string, k ...string) string {
	return f.getValFromImage[i+":"+joinKeys(k)]
}
func (f *fakeDocker) InspectImage(string) (string, error) {
	return f.inspectImageOut, f.inspectImageErr
}
func (f *fakeDocker) LaunchDitKubernetesServers() (string, error) {
	return f.launchK8sOut, f.launchK8sErr
}
func (f *fakeDocker) Pull(string) (string, error) { f.PullCalls++; return "", f.pullErr }
func (f *fakeDocker) Remove(name string, _ bool) (string, error) {
	f.RemoveCalls++
	f.RemovedNames = append(f.RemovedNames, name)
	return "", f.removeErr
}
func (f *fakeDocker) RemoveDitImages(string) (string, error)    { return "", f.removeImagesErr }
func (f *fakeDocker) RemoveVolume(string, bool) (string, error) { return "", f.removeVolumeErr }
func (f *fakeDocker) Tag(string, string) (string, error)        { return "", f.tagErr }
func (f *fakeDocker) Version() (string, error)                  { return "", f.versionErr }
func (f *fakeDocker) VolumeExists(name string) bool             { return f.volumeExists[name] }

func joinKeys(k []string) string {
	out := ""
	for i, s := range k {
		if i > 0 {
			out += "."
		}
		out += s
	}
	return out
}

// fakeK8s satisfies the kubernetesClient interface.
type fakeK8s struct {
	createStatefulSetErr error
	statefulSetStatus    string
	statefulSetStatusErr error

	CreateStatefulSetCalls       int
	WaitForStatefulSetCalls      int
	StartPortForwardingCalls     int
	StopPortForwardingCalls      int
	UpdateStatefulSetVolumesCall int
	DeleteStatefulSpecCalls      int
	StopStatefulSetCalls         int
	StartStatefulSetCalls        int
}

func (f *fakeK8s) CreateStatefulSet(repoName, imageId string, ports []int, volumes []ditclient.Volume, environment []string) error {
	f.CreateStatefulSetCalls++
	return f.createStatefulSetErr
}
func (f *fakeK8s) GetStatefulSetStatus(repoName string) (string, error) {
	return f.statefulSetStatus, f.statefulSetStatusErr
}
func (f *fakeK8s) WaitForStatefulSet(string)  { f.WaitForStatefulSetCalls++ }
func (f *fakeK8s) StartPortForwarding(string) { f.StartPortForwardingCalls++ }
func (f *fakeK8s) StopPortForwarding(string)  { f.StopPortForwardingCalls++ }
func (f *fakeK8s) UpdateStatefulSetVolumes(string, []ditclient.Volume) {
	f.UpdateStatefulSetVolumesCall++
}
func (f *fakeK8s) DeleteStatefulSpec(string) { f.DeleteStatefulSpecCalls++ }
func (f *fakeK8s) StopStatefulSet(string)    { f.StopStatefulSetCalls++ }
func (f *fakeK8s) StartStatefulSet(string)   { f.StartStatefulSetCalls++ }

// with swaps newDocker AND k8s for the duration of fn.
func with(t *testing.T, d dockerClient, ki kubernetesClient, fn func()) {
	t.Helper()
	origDocker := newDocker
	origK8s := k8s
	defer func() {
		newDocker = origDocker
		k8s = origK8s
	}()
	newDocker = func(string, int) dockerClient { return d }
	k8s = ki
	fn()
}
