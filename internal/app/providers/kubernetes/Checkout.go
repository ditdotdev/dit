package kubernetes

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func Checkout(repoName string, guid string, tags []string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	var sourceCommit string
	if guid == "" {
		if len(tags) > 0 {
			commits, _, _ := commitsApi.ListCommits(ctx, repoName).Tag(tags).Execute()
			if len(commits) == 0 {
				fmt.Println("no matching commits found")
				os.Exit(1)
			}
			sourceCommit = commits[0].Id
		} else {
			status, _, _ := repositoriesApi.GetRepositoryStatus(ctx, repoName).Execute()
			if status.GetSourceCommit() == "" {
				fmt.Println("no commits present, run 'd3 commit' first")
				os.Exit(1)
			}
			sourceCommit = status.GetSourceCommit()
		}
	} else {
		if len(tags) > 0 {
			fmt.Println("tags and commit cannot both be specified")
			os.Exit(1)
		}
		sourceCommit = guid
	}

	status, _, _ := commitsApi.GetCommitStatus(ctx, repoName, sourceCommit).Execute()

	if !status.Ready {
		fmt.Println("Waiting for commit to be ready")
		c := true
		for c {
			commitStatus, _, _ := commitsApi.GetCommitStatus(ctx, repoName, sourceCommit).Execute()
			if commitStatus.Ready {
				c = false
			}
			time.Sleep(1000)
		}
	}

	fmt.Println("Checkout " + sourceCommit)
	if _, err := commitsApi.CheckoutCommit(ctx, repoName, sourceCommit).Execute(); err != nil {
		fmt.Printf("Error checking out commit %s: %v\n", sourceCommit, err)
		os.Exit(1)
	}

	fmt.Println("Stopping port forwarding")
	k8s.StopPortForwarding(repoName)

	// Scale to 0 before swapping the StatefulSet's PVC reference.
	// Patching spec.template.spec.volumes[*].persistentVolumeClaim.claimName
	// alone does NOT trigger pod recreation on a StatefulSet — k8s only
	// rolls pods on changes to spec.template.spec.containers (image, env,
	// etc.), not on volume claim name updates. Without the stop/start
	// dance the running pod keeps the OLD PVC mounted, postgres serves
	// the pre-checkout state, and `d3 checkout` returns success without
	// actually restoring data. Surfaced by kubernetes-tests.bats test 19
	// on PR #113 — checkout returned in 2s, table still missing.
	fmt.Println("Stopping deployment to release old PVC")
	k8s.StopStatefulSet(repoName)
	k8s.WaitForStatefulSet(repoName)

	fmt.Println("Updating deployment")
	vols, _, _ := volumesApi.ListVolumes(ctx, repoName).Execute()
	k8s.UpdateStatefulSetVolumes(repoName, vols)

	fmt.Println("Restarting deployment with new PVC")
	k8s.StartStatefulSet(repoName)
	k8s.WaitForStatefulSet(repoName)

	fmt.Println("Starting port forwarding")
	k8s.StartPortForwarding(repoName)
}
