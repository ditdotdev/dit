package kubernetes

import (
	"datadatdat/internal/app/providers/common"
	"fmt"
	"os"
	"strconv"
)

// Status prints runtime and storage status for a repository running in a
// Kubernetes context. The common.Status used by the docker/local provider
// looks up the per-repo container via `docker inspect`, which for a
// kubernetes-backed repo always fails (the workload is a Pod managed by a
// StatefulSet, not a top-level docker container). That fallback would
// therefore print "Status: detached" even for a perfectly healthy repo.
// Here we derive the runtime status from the StatefulSet directly via the
// same logic that `d3 ls` uses, and then print the same header/volume
// layout as common.Status for consistency.
func Status(repo string, port int, context string) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)

	s, resp, err := repositoriesApi.GetRepositoryStatus(ctx, repo)
	if err != nil || (resp != nil && resp.StatusCode >= 400) {
		fmt.Printf("Error: repository '%s' not found\n", repo)
		os.Exit(1)
	}

	runtimeStatus, _ := k8s.GetStatefulSetStatus(repo)
	fmt.Printf("%20s %s\n", "Status: ", runtimeStatus)

	if s.LastCommit != "" {
		fmt.Printf("%20s %s\n", "Last Commit: ", s.LastCommit)
	}
	if s.SourceCommit != "" {
		fmt.Printf("%20s %s\n", "Source Commit: ", s.SourceCommit)
	}

	vols, _, _ := volumesApi.ListVolumes(ctx, repo)
	fmt.Printf("%-30s  %-12s  %s\n", "Volume", "Uncompressed", "Compressed")
	for _, v := range vols {
		vstat, _, _ := volumesApi.GetVolumeStatus(ctx, repo, v.Name)
		path, _ := vstat.Properties["path"].(string)
		fmt.Printf("%-30s  %-12s  %s\n", path,
			common.ByteCountBinary(vstat.LogicalSize),
			common.ByteCountBinary(vstat.ActualSize),
		)
	}
}
