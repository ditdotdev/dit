package clients

import (
	"strings"
	"testing"
	"time"

	datadatdatclient "github.com/datadatdat/datadatdat-client-go"
	v1Apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const testNamespace = "default"

// swapClient replaces the package-level k8s client with a fake for the
// duration of one test. Caller must defer the returned restore func.
func swapClient(t *testing.T, replacement k8s.Interface) func() {
	t.Helper()
	original := client
	client = replacement
	return func() { client = original }
}

// TestDeleteStatefulSpecToleratesMissingStatefulSet covers the case where
// `d3 rm -f` runs against a repository whose StatefulSet never finished
// being created (e.g. an earlier failed `d3 run`). The function should NOT
// panic; a missing StatefulSet is a valid no-op for deletion.
func TestDeleteStatefulSpecToleratesMissingStatefulSet(t *testing.T) {
	restore := swapClient(t, fake.NewSimpleClientset())
	defer restore()

	k := kubernetes{namespace: testNamespace}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("DeleteStatefulSpec panicked on a missing StatefulSet: %v", r)
		}
	}()
	k.DeleteStatefulSpec("nonexistent")
}

// TestDeleteStatefulSpecDeletesExistingResources verifies the happy path —
// when both the StatefulSet and the Service exist, both are removed.
func TestDeleteStatefulSpecDeletesExistingResources(t *testing.T) {
	ns := testNamespace
	statefulSet := &v1Apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "my-repo", Namespace: ns},
	}
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "my-repo", Namespace: ns},
	}
	fakeClient := fake.NewSimpleClientset(statefulSet, service)
	restore := swapClient(t, fakeClient)
	defer restore()

	k := kubernetes{namespace: ns}
	k.DeleteStatefulSpec("my-repo")

	if _, err := fakeClient.AppsV1().StatefulSets(ns).Get(ctx, "my-repo", metav1.GetOptions{}); err == nil {
		t.Errorf("expected StatefulSet to be deleted, still found it")
	}
	if _, err := fakeClient.CoreV1().Services(ns).Get(ctx, "my-repo", metav1.GetOptions{}); err == nil {
		t.Errorf("expected Service to be deleted, still found it")
	}
}

// TestCreateStatefulSetReadsPVCNameFromVolumeConfig covers the case where
// the datadatdat server returns volume metadata with the per-volume PVC
// name in `Config["pvc"]` (its actual server-generated location), not
// `Properties["pvc"]`. The StatefulSet produced must reference the
// server-provided PVC claim name.
func TestCreateStatefulSetReadsPVCNameFromVolumeConfig(t *testing.T) {
	ns := testNamespace
	fakeClient := fake.NewSimpleClientset()
	restore := swapClient(t, fakeClient)
	defer restore()

	k := kubernetes{namespace: ns}

	// Matches the server's actual response shape: path in Properties,
	// pvc name in Config (see /v1/repositories/<repo>/volumes).
	vol := datadatdatclient.Volume{
		Name:       "v0",
		Properties: map[string]interface{}{"path": "/var/lib/postgresql"},
		Config:     map[string]interface{}{"pvc": "5d2810f5-v0", "namespace": "default"},
	}

	err := k.CreateStatefulSet("demo-db", "postgres:latest", []int{5432}, []datadatdatclient.Volume{vol}, nil)
	if err != nil {
		t.Fatalf("CreateStatefulSet failed: %v", err)
	}

	ss, err := fakeClient.AppsV1().StatefulSets(ns).Get(ctx, "demo-db", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("StatefulSet not created: %v", err)
	}
	if len(ss.Spec.Template.Spec.Volumes) != 1 {
		t.Fatalf("expected 1 pod volume, got %d", len(ss.Spec.Template.Spec.Volumes))
	}
	got := ss.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim
	if got == nil {
		t.Fatal("expected PersistentVolumeClaim volume source, got nil")
	}
	if got.ClaimName != "5d2810f5-v0" {
		t.Errorf("ClaimName = %q, want %q (server-provided PVC name from Config)", got.ClaimName, "5d2810f5-v0")
	}
}

// TestWaitForStatefulSetReturnsWhenStatefulSetMissing covers the case
// where d3 stop / d3 start runs against a repository whose StatefulSet
// was never created (or was deleted out-of-band). Pre-fix, WaitForStatefulSet
// busy-looped forever — the BATS suite hit the 10-min CI wall on
// `d3 stop` because GetStatefulSetStatus returned "detached" and that
// wasn't recognized as terminal.
func TestWaitForStatefulSetReturnsWhenStatefulSetMissing(t *testing.T) {
	restore := swapClient(t, fake.NewSimpleClientset())
	defer restore()

	// Use a tiny timeout so the test exits fast if the bug regresses.
	originalTimeout := waitForStatefulSetTimeout
	waitForStatefulSetTimeout = 200 * time.Millisecond
	defer func() { waitForStatefulSetTimeout = originalTimeout }()

	k := kubernetes{namespace: testNamespace}

	done := make(chan struct{})
	go func() {
		defer close(done)
		k.WaitForStatefulSet("nonexistent")
	}()

	select {
	case <-done:
		// success — function returned within the deadline
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForStatefulSet did not return within 2s for a missing StatefulSet (expected to bail after the deadline)")
	}
}

// TestWaitForStatefulSetReturnsWhenRunning covers the happy path: the
// function should return quickly once the StatefulSet's replicas match
// readyReplicas (status="running").
func TestWaitForStatefulSetReturnsWhenRunning(t *testing.T) {
	ns := testNamespace
	replicas := int32(1)
	ss := &v1Apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-db", Namespace: ns},
		Spec: v1Apps.StatefulSetSpec{
			Replicas: &replicas,
		},
		Status: v1Apps.StatefulSetStatus{
			Replicas:        1,
			ReadyReplicas:   1,
			UpdateRevision:  "r1",
			CurrentRevision: "r1",
		},
	}
	restore := swapClient(t, fake.NewSimpleClientset(ss))
	defer restore()

	k := kubernetes{namespace: ns}

	done := make(chan struct{})
	go func() {
		defer close(done)
		k.WaitForStatefulSet("demo-db")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForStatefulSet did not return within 2s for a running StatefulSet")
	}
}

// TestPortFromPidFilename covers the small parser that recovers the
// forwarded port from a pid file path. Used by StopPortForwarding's
// "find listening pid by port" fallback.
func TestPortFromPidFilename(t *testing.T) {
	cases := []struct {
		in        string
		wantPort  int
		wantValid bool
	}{
		{"portforward-demo-db-5432.pid", 5432, true},
		{"/c/Users/x/.datadatdat/portforward-my-repo-8080.pid", 8080, true},
		{"portforward-repo-with-dashes-443.pid", 443, true},
		{"portforward-demo-db.pid", 0, false}, // missing port
		{"portforward-demo-db-notanum.pid", 0, false},
		{"unrelated-file.txt", 0, false},
	}
	for _, c := range cases {
		got, ok := portFromPidFilename(c.in)
		if ok != c.wantValid {
			t.Errorf("portFromPidFilename(%q) valid = %v, want %v", c.in, ok, c.wantValid)
		}
		if got != c.wantPort {
			t.Errorf("portFromPidFilename(%q) port = %d, want %d", c.in, got, c.wantPort)
		}
	}
}

// TestGetStatefulSetStatusReportsRunning covers the happy path: when the
// StatefulSet has replicas=ready=1, status must be "running" (not "detached"
// as the docker-centric common.Status fallback returned on the k8s path).
func TestGetStatefulSetStatusReportsRunning(t *testing.T) {
	ns := testNamespace
	replicas := int32(1)
	ss := &v1Apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-db", Namespace: ns},
		Spec: v1Apps.StatefulSetSpec{
			Replicas: &replicas,
		},
		Status: v1Apps.StatefulSetStatus{
			Replicas:        1,
			ReadyReplicas:   1,
			UpdateRevision:  "r1",
			CurrentRevision: "r1",
		},
	}
	fakeClient := fake.NewSimpleClientset(ss)
	restore := swapClient(t, fakeClient)
	defer restore()

	k := kubernetes{namespace: ns}
	status, err := k.GetStatefulSetStatus("demo-db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != statusRunning {
		t.Errorf("status = %q, want \"running\"", status)
	}
}

// TestGetStatefulSetStatusWaitsForObservedGeneration covers the race that
// caused `kubectl exec mydb-0` to fail with "pod does not have a host
// assigned" right after `d3 checkout`. The flow:
//
//  1. Checkout scales the StatefulSet to 0 (Stop), waits.
//  2. Checkout patches the PVC, then scales back to 1 (Start), waits.
//  3. Wait/StartPortForwarding return; user runs kubectl exec.
//
// At step 2, `Status.Replicas` and `Status.ReadyReplicas` are still the
// pre-patch values (0/0) for a brief window because the StatefulSet
// controller hasn't observed the new generation yet. Pre-fix,
// GetStatefulSetStatus saw Replicas==0 and returned "stopped" (terminal),
// so WaitForStatefulSet exited immediately — before the pod was
// scheduled. The fix is to require Status.ObservedGeneration >=
// metadata.Generation before trusting any replicas-derived status.
func TestGetStatefulSetStatusWaitsForObservedGeneration(t *testing.T) {
	ns := testNamespace
	replicas := int32(1)
	ss := &v1Apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-db", Namespace: ns, Generation: 2},
		Spec: v1Apps.StatefulSetSpec{
			Replicas: &replicas,
		},
		Status: v1Apps.StatefulSetStatus{
			ObservedGeneration: 1,
			Replicas:           0,
			ReadyReplicas:      0,
			UpdateRevision:     "r1",
			CurrentRevision:    "r1",
		},
	}
	restore := swapClient(t, fake.NewSimpleClientset(ss))
	defer restore()

	k := kubernetes{namespace: ns}
	status, err := k.GetStatefulSetStatus("demo-db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Anything terminal (running, stopped, detached) is wrong here —
	// WaitForStatefulSet would return before the rollout starts.
	if status == statusRunning || status == statusStopped || status == statusDetached {
		t.Errorf("status = %q for stale (unobserved) generation; expected a non-terminal state like %q", status, statusStarting)
	}
}

// TestCreateStatefulSetDetectsOrphanedService covers the case where a
// previous d3 session left a Service of the same name behind (e.g. user
// Ctrl-C'd before `d3 rm`). The next `d3 run -n <repo>` must not surface
// the raw k8s `services "<repo>" already exists` error; it must fail fast
// with a recovery hint that tells the user how to clean up. See issue #126.
func TestCreateStatefulSetDetectsOrphanedService(t *testing.T) {
	ns := testNamespace
	existingService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "mydb", Namespace: ns},
	}
	fakeClient := fake.NewSimpleClientset(existingService)
	restore := swapClient(t, fakeClient)
	defer restore()

	k := kubernetes{namespace: ns}
	vol := datadatdatclient.Volume{
		Name:       "v0",
		Properties: map[string]interface{}{"path": "/var/lib/postgresql"},
		Config:     map[string]interface{}{"pvc": "pvc-v0"},
	}

	err := k.CreateStatefulSet("mydb", "postgres:latest", []int{5432}, []datadatdatclient.Volume{vol}, nil)
	if err == nil {
		t.Fatal("expected CreateStatefulSet to fail when a Service of the same name already exists, got nil")
	}
	msg := err.Error()
	// Recovery-hint shape from issue #126.
	for _, substr := range []string{
		"mydb",
		ns,
		"service/mydb",
		"d3 rm",
		"datadatdatRepository=mydb",
	} {
		if !strings.Contains(msg, substr) {
			t.Errorf("error message missing %q; got:\n%s", substr, msg)
		}
	}
	// Must NOT be the raw k8s "already exists" surface — that's the bug.
	if strings.Contains(msg, "services \"mydb\" already exists") {
		t.Errorf("error leaks raw k8s AlreadyExists surface; got:\n%s", msg)
	}
}

// TestCreateStatefulSetDetectsOrphanedStatefulSet covers the symmetric
// case where the StatefulSet survived but the Service did not. Issue #126.
func TestCreateStatefulSetDetectsOrphanedStatefulSet(t *testing.T) {
	ns := testNamespace
	existing := &v1Apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "mydb", Namespace: ns},
	}
	fakeClient := fake.NewSimpleClientset(existing)
	restore := swapClient(t, fakeClient)
	defer restore()

	k := kubernetes{namespace: ns}
	vol := datadatdatclient.Volume{
		Name:       "v0",
		Properties: map[string]interface{}{"path": "/var/lib/postgresql"},
		Config:     map[string]interface{}{"pvc": "pvc-v0"},
	}

	err := k.CreateStatefulSet("mydb", "postgres:latest", []int{5432}, []datadatdatclient.Volume{vol}, nil)
	if err == nil {
		t.Fatal("expected CreateStatefulSet to fail when a StatefulSet of the same name already exists, got nil")
	}
	if !strings.Contains(err.Error(), "statefulset/mydb") {
		t.Errorf("error message missing \"statefulset/mydb\"; got:\n%s", err.Error())
	}
}

// TestCreateStatefulSetDetectsOrphanedPVC covers the case where the
// StatefulSet/Service were cleaned up but PVCs labelled with the d3
// repository remain. Issue #126.
func TestCreateStatefulSetDetectsOrphanedPVC(t *testing.T) {
	ns := testNamespace
	existingPVC := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v0-mydb-0",
			Namespace: ns,
			Labels:    map[string]string{labelDatadatdatRepository: "mydb"},
		},
	}
	fakeClient := fake.NewSimpleClientset(existingPVC)
	restore := swapClient(t, fakeClient)
	defer restore()

	k := kubernetes{namespace: ns}
	vol := datadatdatclient.Volume{
		Name:       "v0",
		Properties: map[string]interface{}{"path": "/var/lib/postgresql"},
		Config:     map[string]interface{}{"pvc": "pvc-v0"},
	}

	err := k.CreateStatefulSet("mydb", "postgres:latest", []int{5432}, []datadatdatclient.Volume{vol}, nil)
	if err == nil {
		t.Fatal("expected CreateStatefulSet to fail when a labelled PVC for the repo already exists, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "persistentvolumeclaim/v0-mydb-0") {
		t.Errorf("error message missing \"persistentvolumeclaim/v0-mydb-0\"; got:\n%s", msg)
	}
	if !strings.Contains(msg, "kubectl delete") {
		t.Errorf("error message missing \"kubectl delete\" recovery hint; got:\n%s", msg)
	}
}

// TestCreateStatefulSetIgnoresPVCWithoutLabel covers a non-d3 PVC happening
// to share the namespace. We must NOT block on PVCs that don't carry the
// datadatdatRepository=<repo> label; otherwise random user PVCs would fail
// d3 runs. Happy path with no orphans should succeed.
func TestCreateStatefulSetIgnoresPVCWithoutLabel(t *testing.T) {
	ns := testNamespace
	unrelatedPVC := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-other-pvc",
			Namespace: ns,
		},
	}
	fakeClient := fake.NewSimpleClientset(unrelatedPVC)
	restore := swapClient(t, fakeClient)
	defer restore()

	k := kubernetes{namespace: ns}
	vol := datadatdatclient.Volume{
		Name:       "v0",
		Properties: map[string]interface{}{"path": "/var/lib/postgresql"},
		Config:     map[string]interface{}{"pvc": "pvc-v0"},
	}

	err := k.CreateStatefulSet("mydb", "postgres:latest", []int{5432}, []datadatdatclient.Volume{vol}, nil)
	if err != nil {
		t.Fatalf("CreateStatefulSet should succeed when no d3-labelled resources exist; got: %v", err)
	}
}

// TestDeleteStatefulSpecToleratesMissingService covers the case where the
// StatefulSet exists but the Service was already deleted (or never created).
// Service deletion must not panic.
func TestDeleteStatefulSpecToleratesMissingService(t *testing.T) {
	ns := testNamespace
	statefulSet := &v1Apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "my-repo", Namespace: ns},
	}
	fakeClient := fake.NewSimpleClientset(statefulSet)
	restore := swapClient(t, fakeClient)
	defer restore()

	k := kubernetes{namespace: ns}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("DeleteStatefulSpec panicked on a missing Service: %v", r)
		}
	}()
	k.DeleteStatefulSpec("my-repo")
}
