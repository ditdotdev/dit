package local

import (
	"datadatdat/internal/app"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"

	"bytes"
	"io"
)

// captureStdout captures stdout during a function call and returns the output.
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

// startMockServer spins up an httptest server and returns its port number.
func startMockServer(t *testing.T, handler http.HandlerFunc) int {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	p, _ := strconv.Atoi(u.Port())
	return p
}

type exitPanic struct{ code int }

// captureExit runs fn with osExit swapped to a recorder.
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

// fakeDocker is a fully-stubbed dockerClient that records calls and lets each
// test customize return values for the methods it cares about.
type fakeDocker struct {
	// Inputs/outputs the test wants to control:
	containerExists              map[string]bool
	containerIsRunning           map[string]bool
	containerExistsErr           error
	cpErr                        error
	createVolumeErr              error
	datadatdatLatestIsDownloaded bool
	datadatdatLaunchAvailable    bool
	datadatdatServerAvailable    bool
	fetchLaunchLogs              []string
	formatVolumeName             func(string, string) string
	identity                     string
	getSliceFromContainer        map[string][]string
	getSliceFromImage            map[string][]string
	getValFromContainer          map[string]string
	getValFromContainerErrs      map[string]error
	getValFromImage              map[string]string
	inspectContainerOut          string
	inspectContainerErr          error
	inspectImageOut              string
	inspectImageErr              error
	launchOut                    string
	launchErr                    error
	listVolumes                  []string
	pullErr                      error
	removeErr                    error
	removeDatadatdatImagesErr    error
	removeDatadatdatLaunchErr    error
	removeDatadatdatServerErr    error
	removeDatadatdatVolumeErr    error
	removeStoppedErr             error
	removeVolumeErr              error
	runOut                       string
	runErr                       error
	startErr                     error
	stopErr                      error
	tagErr                       error
	teardownErr                  error
	versionErr                   error
	volumeExists                 map[string]bool

	// Call counts to assert on:
	StopCalls, StartCalls, RunCalls int
	RemoveCalls                     int
	PullCalls                       int
}

func (f *fakeDocker) ContainerExists(c string) (bool, error) {
	return f.containerExists[c], f.containerExistsErr
}
func (f *fakeDocker) ContainerIsRunning(c string) (bool, error) {
	return f.containerIsRunning[c], nil
}
func (f *fakeDocker) Cp(s, t string) (string, error)           { return "", f.cpErr }
func (f *fakeDocker) CreateVolume(n, p string) (string, error) { return "", f.createVolumeErr }
func (f *fakeDocker) DatadatdatLatestIsDownloaded(string, app.Version) bool {
	return f.datadatdatLatestIsDownloaded
}
func (f *fakeDocker) DatadatdatLaunchIsAvailable() (bool, error) {
	return f.datadatdatLaunchAvailable, nil
}
func (f *fakeDocker) DatadatdatServerIsAvailable() (bool, error) {
	return f.datadatdatServerAvailable, nil
}
func (f *fakeDocker) FetchLaunchLogs() []string { return f.fetchLaunchLogs }
func (f *fakeDocker) FormatVolumeName(repo, vol string) string {
	if f.formatVolumeName != nil {
		return f.formatVolumeName(repo, vol)
	}
	return repo + "_" + vol
}
func (f *fakeDocker) GetIdentity() string { return f.identity }
func (f *fakeDocker) GetSliceFromContainer(c string, key ...string) []string {
	return f.getSliceFromContainer[c]
}
func (f *fakeDocker) GetSliceFromImage(i string, key ...string) []string {
	return f.getSliceFromImage[i+":"+joinKeys(key)]
}
func (f *fakeDocker) GetValFromContainer(c string, key ...string) (string, error) {
	k := c + ":" + joinKeys(key)
	return f.getValFromContainer[k], f.getValFromContainerErrs[k]
}
func (f *fakeDocker) GetValFromImage(i string, key ...string) string {
	return f.getValFromImage[i+":"+joinKeys(key)]
}
func (f *fakeDocker) InspectContainer(string) (string, error) {
	return f.inspectContainerOut, f.inspectContainerErr
}
func (f *fakeDocker) InspectImage(string) (string, error) {
	return f.inspectImageOut, f.inspectImageErr
}
func (f *fakeDocker) LaunchDatadatdatServers() (string, error) { return f.launchOut, f.launchErr }
func (f *fakeDocker) ListVolumes(string) []string              { return f.listVolumes }
func (f *fakeDocker) Pull(string) (string, error)              { f.PullCalls++; return "", f.pullErr }
func (f *fakeDocker) Remove(string, bool) (string, error)      { f.RemoveCalls++; return "", f.removeErr }
func (f *fakeDocker) RemoveDatadatdatImages(string) (string, error) {
	return "", f.removeDatadatdatImagesErr
}
func (f *fakeDocker) RemoveDatadatdatLaunch() (string, error) { return "", f.removeDatadatdatLaunchErr }
func (f *fakeDocker) RemoveDatadatdatServer() (string, error) { return "", f.removeDatadatdatServerErr }
func (f *fakeDocker) RemoveDatadatdatVolume() (string, error) { return "", f.removeDatadatdatVolumeErr }
func (f *fakeDocker) RemoveStopped(string) (string, error)    { return "", f.removeStoppedErr }
func (f *fakeDocker) RemoveVolume(string, bool) (string, error) {
	return "", f.removeVolumeErr
}
func (f *fakeDocker) Run(string, string, []string) (string, error) {
	f.RunCalls++
	return f.runOut, f.runErr
}
func (f *fakeDocker) Start(string) (string, error)               { f.StartCalls++; return "", f.startErr }
func (f *fakeDocker) Stop(string) (string, error)                { f.StopCalls++; return "", f.stopErr }
func (f *fakeDocker) Tag(string, string) (string, error)         { return "", f.tagErr }
func (f *fakeDocker) TeardownDatadatdatServers() (string, error) { return "", f.teardownErr }
func (f *fakeDocker) Version() (string, error)                   { return "", f.versionErr }
func (f *fakeDocker) VolumeExists(name string) bool              { return f.volumeExists[name] }

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

// withDocker swaps both newDocker and newDockerWithRegistry to the supplied
// fake for the duration of fn.
func withDocker(t *testing.T, d dockerClient, fn func()) {
	t.Helper()
	originalNew := newDocker
	originalNewWithReg := newDockerWithRegistry
	defer func() {
		newDocker = originalNew
		newDockerWithRegistry = originalNewWithReg
	}()
	newDocker = func(string, int) dockerClient { return d }
	newDockerWithRegistry = func(string, int, string) dockerClient { return d }
	fn()
}
