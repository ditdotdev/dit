package kubernetes

import "testing"

// Test the SetOsExitForTesting + UseNoopK8sForTesting seams from inside
// the package so they're counted toward this package's coverage. The
// parent providers package also uses them but cross-package coverage
// requires -coverpkg which isn't on the default `go test ./...` path.

func TestSetOsExitForTesting_RoundTrip(t *testing.T) {
	called := false
	prev := SetOsExitForTesting(func(int) { called = true })
	defer SetOsExitForTesting(prev)

	osExit(7)
	if !called {
		t.Error("SetOsExitForTesting did not install the swap")
	}
}

func TestUseNoopK8sForTesting_RoundTrip(t *testing.T) {
	restore := UseNoopK8sForTesting()
	defer restore()

	// All noopK8s methods are no-ops; the goal is to exercise their
	// bodies for coverage. Each method takes positional args that the
	// signature requires.
	k8s.WaitForStatefulSet("repo")
	k8s.StartPortForwarding("repo")
	k8s.StopPortForwarding("repo")
	k8s.UpdateStatefulSetVolumes("repo", nil)
	k8s.DeleteStatefulSpec("repo")
	k8s.StopStatefulSet("repo")
	k8s.StartStatefulSet("repo")
	if status, err := k8s.GetStatefulSetStatus("repo"); status != "running" || err != nil {
		t.Errorf("expected (running, nil), got (%s, %v)", status, err)
	}
	if err := k8s.CreateStatefulSet("repo", "img", nil, nil, nil); err != nil {
		t.Errorf("unexpected CreateStatefulSet error: %v", err)
	}
}
