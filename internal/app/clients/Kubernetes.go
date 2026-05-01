package clients

import (
	"context"
	"errors"
	"fmt"
	datadatdatclient "github.com/datadatdat/datadatdat-client-go"
	v1Apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var ctx = context.Background()

// labelDatadatdatRepository is the kubernetes label key used to associate
// resources (StatefulSets, Services, etc.) with their owning d3 repository.
const labelDatadatdatRepository = "datadatdatRepository"

var client k8s.Interface

type kubernetes struct {
	namespace string
	host      string
	port      int
}

func Kubernetes(n string, h string, p int) kubernetes {
	return kubernetes{
		namespace: n,
		host:      h,
		port:      p,
	}
}

func init() {
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err == nil {
		client, _ = k8s.NewForConfig(config)
	}
}

/**
 * For our repositories, we keep it very simple. There is a single headless service that is responsible for exposing
 * the ports in the container. We then create a single replica stateful set with the given volumes (each with
 * existing PVCs) mapped in.
 */
func (k kubernetes) CreateStatefulSet(repoName string, imageId string, ports []int, volumes []datadatdatclient.Volume, environment []string) error {
	var err error
	objectMeta := metav1.ObjectMeta{
		Name:      repoName,
		Namespace: k.namespace,
		Labels:    map[string]string{labelDatadatdatRepository: repoName},
	}
	servicePorts := make([]v1.ServicePort, 0, len(ports))
	for _, port := range ports {
		servicePorts = append(servicePorts, v1.ServicePort{
			Name: "port-" + strconv.Itoa(port),
			// #nosec G115 -- Port numbers are bounded to 0-65535, safe to convert to int32
			Port: int32(port),
		})
	}
	serviceSpec := v1.ServiceSpec{
		Ports:     servicePorts,
		Selector:  map[string]string{labelDatadatdatRepository: repoName},
		ClusterIP: "None",
	}
	service := v1.Service{
		ObjectMeta: objectMeta,
		Spec:       serviceSpec,
		Status:     v1.ServiceStatus{},
	}
	createMetadata := metav1.CreateOptions{
		DryRun:       nil,
		FieldManager: "",
	}
	_, err = client.CoreV1().Services(k.namespace).Create(ctx, &service, createMetadata)
	if err != nil {
		return err
	}

	containerPorts := make([]v1.ContainerPort, 0, len(ports))
	for _, port := range ports {
		containerPorts = append(containerPorts, v1.ContainerPort{
			Name: "port-" + strconv.Itoa(port),
			// #nosec G115 -- Port numbers are bounded to 0-65535, safe to convert to int32
			ContainerPort: int32(port),
		})
	}
	envs := make([]v1.EnvVar, 0, len(environment))
	for _, environment := range environment {
		s := strings.Split(environment, "=")
		envs = append(envs, v1.EnvVar{
			Name:  s[0],
			Value: s[1],
		})
	}
	volumeMounts := make([]v1.VolumeMount, 0, len(volumes))
	for _, volume := range volumes {
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      volume.Name,
			MountPath: volume.Properties["path"].(string),
		})
	}
	container := v1.Container{
		Name:         repoName,
		Image:        imageId,
		Ports:        containerPorts,
		Env:          envs,
		VolumeMounts: volumeMounts,
	}
	containers := []v1.Container{container}

	vols := make([]v1.Volume, 0, len(volumes))
	for _, volume := range volumes {
		// The server-generated PVC name lives in `Config`, not `Properties`.
		// `Properties` carries client-provided metadata (e.g. mount path);
		// `Config` carries server-generated details (pvc, namespace, size).
		// See /v1/repositories/<repo>/volumes response from the server.
		claimName, ok := volume.Config["pvc"].(string)
		if !ok || claimName == "" {
			return fmt.Errorf("volume %q has no PVC name in Config; server did not populate Config[\"pvc\"]", volume.Name)
		}
		pvc := v1.PersistentVolumeClaimVolumeSource{
			ClaimName: claimName,
		}
		vols = append(vols, v1.Volume{
			Name: volume.Name,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &pvc,
			},
		})
	}
	podSpec := v1.PodSpec{
		Volumes:    vols,
		Containers: containers,
	}
	podTemplate := v1.PodTemplateSpec{
		ObjectMeta: objectMeta,
		Spec:       podSpec,
	}
	replica := int32(1)
	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{labelDatadatdatRepository: repoName},
	}
	statefulSpecs := v1Apps.StatefulSetSpec{
		Replicas:    &replica,
		Selector:    &selector,
		Template:    podTemplate,
		ServiceName: repoName,
	}
	statefulSet := v1Apps.StatefulSet{
		ObjectMeta: objectMeta,
		Spec:       statefulSpecs,
	}
	_, err = client.AppsV1().StatefulSets(k.namespace).Create(ctx, &statefulSet, createMetadata)
	if err != nil {
		return err
	}
	return nil
}

/**
 * Gets the status of a stateful set. We use the following:
 *
 *      detached        No such statefulset present (user deleted it)
 *      updating        Update revision doesn't match current revision
 *      stopped         Number of replicas is 0
 *      running         Number of replicas and ready replicas is 1
 *      failed          Terminal condition prevented stateful set from starting
 *      starting        Number of replicas is 1 but ready replicas is 0
 *
 * We also return a pair, with the second element providing additional context for the "failed" state
 */
func (k kubernetes) GetStatefulSetStatus(repoName string) (string, error) {
	set, err := client.AppsV1().StatefulSets(k.namespace).Get(ctx, repoName, metav1.GetOptions{})
	if err != nil {
		// Return "detached" for 404 errors, or propagate other errors
		return "detached", err
	}
	if set == nil {
		return "unknown", nil
	}
	if set.Status.UpdateRevision != set.Status.CurrentRevision {
		return "update", nil
	}
	if set.Status.Replicas == 0 {
		return "stopped", nil
	}
	if set.Status.Replicas == set.Status.ReadyReplicas {
		return "running", nil
	}
	pod, err := client.CoreV1().Pods(k.namespace).Get(ctx, repoName, metav1.GetOptions{})
	if err != nil {
		// If pod doesn't exist, return starting state
		return "starting", nil
	}
	conditions := pod.Status.Conditions
	for _, condition := range conditions {
		if condition.Reason == "Unschedulable" {
			return "failed", errors.New("Pod failed to be scheduled: " + condition.Message)
		}
	}
	return "starting", nil
}

// waitForStatefulSetTimeout caps how long WaitForStatefulSet will busy-poll
// before giving up. Exposed as a package var (not a const) so tests can shrink
// it to keep the suite fast.
var waitForStatefulSetTimeout = 2 * time.Minute

// waitForStatefulSetPollInterval is the gap between status polls. Pre-fix this
// was `time.Sleep(1000)` which is 1000 nanoseconds — effectively a busy loop.
var waitForStatefulSetPollInterval = 1 * time.Second

/**
 * Wait for the given statefulset to reach a terminal state (running or stopped), throwing an error if we've
 * reached the failed state. Bails after waitForStatefulSetTimeout if the StatefulSet never reaches a terminal
 * state — a "detached" status (no StatefulSet present) is treated as terminal so callers like d3 stop / d3 rm
 * don't hang forever waiting for resources that were never created.
 */
func (k kubernetes) WaitForStatefulSet(repoName string) {
	deadline := time.Now().Add(waitForStatefulSetTimeout)
	for {
		status, err := k.GetStatefulSetStatus(repoName)
		switch status {
		case "failed":
			panic(err)
		case "running", "stopped", "detached":
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(waitForStatefulSetPollInterval)
	}
}

/**
 * Forward port for a container. For now, we're using a temporary solution of launching 'kubectl-forward' in the
 * background. This is totally brittle, as the commands will fail in the background as pods are stopped and
 * connections broken. And if you restart the host system, there is no way to restart them. But it's a quick
 * hack to demonstrate the desired experience until we can build out a more full-featured port forwarder, such
 * as: https://github.com/pixel-point/kube-forwarder
 */
func (k kubernetes) StartPortForwarding(repoName string) {
	// Small grace period: the pod can report Ready before the service
	// endpoint is actually routable. (time.Sleep takes a Duration; a bare
	// literal is nanoseconds, so we explicitly use time.Second here.)
	time.Sleep(1 * time.Second)
	service, _ := client.CoreV1().Services(k.namespace).Get(ctx, repoName, metav1.GetOptions{})
	ports := service.Spec.Ports
	for _, port := range ports {
		// Launch kubectl port-forward as a detached child that outlives the
		// current `d3` invocation. The earlier approach shelled out to
		// `sh -c "... &"` via ce.Exec which waits for the shell; once the
		// shell exits, the `&`-backgrounded grandchild is orphaned and, on
		// Windows, gets reaped almost immediately, so `psql -h localhost`
		// would fail with "connection refused".
		//
		// exec.Command + Start (without Wait) leaves the child running
		// and independent of d3.
		cmd := exec.Command("kubectl", "port-forward", "svc/"+repoName, fmt.Sprint(port.Port)) // #nosec G204 -- repoName and port come from the user's own repo and service spec
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.SysProcAttr = detachedSysProcAttr()
		if err := cmd.Start(); err != nil {
			fmt.Printf("Warning: Failed to setup port forward for port %d: %v\n", port.Port, err)
			continue
		}
		pid := cmd.Process.Pid
		// Release the handle so the child is not waited on by d3's exit.
		if err := cmd.Process.Release(); err != nil {
			fmt.Printf("Warning: Failed to detach port-forward for port %d: %v\n", port.Port, err)
		}
		// Persist the PID so StopPortForwarding can find the process on
		// any OS without depending on `ps | grep` semantics (which Git
		// Bash can't use to see native Windows processes).
		if err := writePortForwardPid(repoName, port.Port, pid); err != nil {
			fmt.Printf("Warning: Failed to record port-forward pid for port %d: %v\n", port.Port, err)
		}
	}
}

// StopPortForwarding kills any kubectl port-forward processes that
// StartPortForwarding launched for this repo. Matches by PID file rather
// than by Service spec lookup so it still works after the Service has
// been deleted (e.g. during `d3 rm`).
//
// Note on Windows: when kubectl is installed via Chocolatey, /c/bin/kubectl
// is a "shim" PE that launches the real kubectl.exe and exits. Go's
// cmd.Process.Pid captures the shim, which is already gone by the time we
// try to kill it — the actual kubectl listening on the local port has a
// different pid. So we also resolve the pid by walking the pid-file's
// port back to whoever is currently LISTENING on it, and kill that too.
func (k kubernetes) StopPortForwarding(repoName string) {
	for _, pidFile := range portForwardPidFilesFor(repoName) {
		killPidFromFile(pidFile)
		if port, ok := portFromPidFilename(pidFile); ok {
			if pid := findListeningPidOnPort(port); pid != 0 {
				if proc, err := os.FindProcess(pid); err == nil {
					_ = proc.Kill()
				}
			}
		}
		_ = os.Remove(pidFile)
	}
}

func killPidFromFile(pidFile string) {
	data, err := os.ReadFile(pidFile) // #nosec G304 -- path is derived from user's home and a known prefix
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return
	}
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill()
	}
}

// portFromPidFilename parses the port number out of a pid file path with
// the shape portforward-<repo>-<port>.pid.
func portFromPidFilename(pidFile string) (int, bool) {
	base := filepath.Base(pidFile)
	if !strings.HasSuffix(base, ".pid") {
		return 0, false
	}
	// Strip suffix and leading "portforward-"
	stem := strings.TrimSuffix(base, ".pid")
	// Port is after the last "-"
	lastDash := strings.LastIndex(stem, "-")
	if lastDash < 0 {
		return 0, false
	}
	port, err := strconv.Atoi(stem[lastDash+1:])
	if err != nil {
		return 0, false
	}
	return port, true
}

// findListeningPidOnPort returns the pid of the process currently bound to
// the given TCP port on the local host, or 0 if nothing is listening.
// Cross-platform via OS-specific lookup (netstat on Windows, lsof elsewhere).
func findListeningPidOnPort(port int) int {
	if runtime.GOOS == "windows" {
		// #nosec G204 -- port is an int rendered via strconv.Itoa, only digits, no shell injection surface.
		out, err := exec.Command("cmd", "/c", "netstat -ano | findstr :"+strconv.Itoa(port)).CombinedOutput()
		if err != nil {
			return 0
		}
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.Contains(line, "LISTENING") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}
			// The local address is fields[1]; only match if its port
			// segment is exactly our port (avoids matching 54321 etc.)
			localAddr := fields[1]
			colon := strings.LastIndex(localAddr, ":")
			if colon < 0 || localAddr[colon+1:] != strconv.Itoa(port) {
				continue
			}
			if pid, err := strconv.Atoi(fields[len(fields)-1]); err == nil {
				return pid
			}
		}
		return 0
	}
	// #nosec G204 -- port is an int rendered via strconv.Itoa, only digits, no shell injection surface.
	out, err := exec.Command("lsof", "-t", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN").CombinedOutput()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
			return pid
		}
	}
	return 0
}

func portForwardPidDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".datadatdat")
}

func portForwardPidFilePath(repoName string, port int32) string {
	return filepath.Join(portForwardPidDir(), "portforward-"+repoName+"-"+fmt.Sprint(port)+".pid")
}

func writePortForwardPid(repoName string, port int32, pid int) error {
	dir := portForwardPidDir()
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	return os.WriteFile(portForwardPidFilePath(repoName, port), []byte(strconv.Itoa(pid)), 0600)
}

func portForwardPidFilesFor(repoName string) []string {
	entries, err := os.ReadDir(portForwardPidDir())
	if err != nil {
		return nil
	}
	prefix := "portforward-" + repoName + "-"
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".pid") {
			out = append(out, filepath.Join(portForwardPidDir(), name))
		}
	}
	return out
}

/**
 * Update the volumes within a given StatefulSet.
 */
func (k kubernetes) UpdateStatefulSetVolumes(repoName string, volumes []datadatdatclient.Volume) {
	// Build a JSONPatch document. Two pre-existing bugs:
	//
	//  1. The previous string-concat used `\\\"` (which evaluates to `\"`,
	//     an escaped backslash + quote) where it should have used `\"`
	//     (which evaluates to `"`). The k8s apiserver rejected the patch
	//     with "error decoding patch: invalid character '\\' looking for
	//     beginning of object key string", and the warning was ignored
	//     by the caller — so the volumes were never updated.
	//
	//  2. JSONPatchType requires an array of operations: `[{...},{...}]`.
	//     The previous code emitted bare `{...}{...}` with no wrapping
	//     brackets and no comma separator, which would have failed to
	//     parse even if the quotes were right.
	//
	// Surfaced by kubernetes-tests.bats test 19 — d3 checkout returned
	// success without actually swapping PVCs because the patch silently
	// failed. See StopStatefulSet/StartStatefulSet patches for the
	// correct shape (array, single-escaped quotes).
	set, _ := client.AppsV1().StatefulSets(k.namespace).Get(ctx, repoName, metav1.GetOptions{})
	specVolumes := set.Spec.Template.Spec.Volumes
	if len(specVolumes) == 0 {
		return
	}
	var ops []string
	for volumeIdx, volumeDef := range specVolumes {
		for _, vol := range volumes {
			if vol.Name == volumeDef.Name {
				pvc, ok := vol.Config["pvc"].(string)
				if !ok || pvc == "" {
					continue
				}
				ops = append(ops, fmt.Sprintf(
					`{"op":"replace","path":"/spec/template/spec/volumes/%d/persistentVolumeClaim/claimName","value":%q}`,
					volumeIdx, pvc,
				))
			}
		}
	}
	if len(ops) == 0 {
		return
	}
	patch := []byte("[" + strings.Join(ops, ",") + "]")
	if _, err := client.AppsV1().StatefulSets(k.namespace).Patch(ctx, repoName, types.JSONPatchType, patch, metav1.PatchOptions{}); err != nil {
		fmt.Printf("Warning: Failed to patch stateful set volumes: %v\n", err)
	}
}

func (k kubernetes) DeleteStatefulSpec(repoName string) {
	// Tolerate NotFound for both the StatefulSet and Service. A repository
	// record can exist on the datadatdat server with no underlying k8s
	// resources if an earlier `d3 run` failed after CreateRepository but
	// before CreateStatefulSet succeeded; `d3 rm -f` must still be able to
	// clean that up without panicking.
	if err := client.AppsV1().StatefulSets(k.namespace).Delete(ctx, repoName, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
		panic(err)
	}
	if err := client.CoreV1().Services(k.namespace).Delete(ctx, repoName, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
		panic(err)
	}
}

/**
 * Stops a stateful set. This is equivalent to setting the number of replicas to zero and waiting for the
 * deployment to update. Its up to callers to wait for the changes to take effect.
 */
func (k kubernetes) StopStatefulSet(repoName string) {
	patch := []byte("[{\"op\":\"replace\",\"path\":\"/spec/replicas\",\"value\":0}]")
	if _, err := client.AppsV1().StatefulSets(k.namespace).Patch(ctx, repoName, types.JSONPatchType, patch, metav1.PatchOptions{}); err != nil {
		fmt.Printf("Warning: Failed to stop stateful set: %v\n", err)
	}
}

/**
 * Opposite of the above, set the number of replicas to one.
 */
func (k kubernetes) StartStatefulSet(repoName string) {
	patch := []byte("[{\"op\":\"replace\",\"path\":\"/spec/replicas\",\"value\":1}]")
	if _, err := client.AppsV1().StatefulSets(k.namespace).Patch(ctx, repoName, types.JSONPatchType, patch, metav1.PatchOptions{}); err != nil {
		fmt.Printf("Warning: Failed to start stateful set: %v\n", err)
	}
}
