package clients

import (
	"testing"

	datadatdatclient "github.com/datadatdat/datadatdat-client-go"
	v1Apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

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

	k := kubernetes{namespace: "default"}

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
	ns := "default"
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
	ns := "default"
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
	got := ss.Spec.Template.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim
	if got == nil {
		t.Fatal("expected PersistentVolumeClaim volume source, got nil")
	}
	if got.ClaimName != "5d2810f5-v0" {
		t.Errorf("ClaimName = %q, want %q (server-provided PVC name from Config)", got.ClaimName, "5d2810f5-v0")
	}
}

// TestGetStatefulSetStatusReportsRunning covers the happy path: when the
// StatefulSet has replicas=ready=1, status must be "running" (not "detached"
// as the docker-centric common.Status fallback returned on the k8s path).
func TestGetStatefulSetStatusReportsRunning(t *testing.T) {
	ns := "default"
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
	if status != "running" {
		t.Errorf("status = %q, want \"running\"", status)
	}
}

// TestDeleteStatefulSpecToleratesMissingService covers the case where the
// StatefulSet exists but the Service was already deleted (or never created).
// Service deletion must not panic.
func TestDeleteStatefulSpecToleratesMissingService(t *testing.T) {
	ns := "default"
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
