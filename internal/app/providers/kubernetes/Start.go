// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"fmt"
	"strconv"
)

func Start(repoName string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	repo, _, _ := repositoriesApi.GetRepository(ctx, repoName).Execute()
	fmt.Println("Updating deployment")
	k8s.StartStatefulSet(repoName)

	fmt.Println("Waiting for deployment to be ready")
	k8s.WaitForStatefulSet(repoName)

	if !disablePortMappingFromRepo(*repo) {
		fmt.Println("Starting port forwarding")
		k8s.StartPortForwarding(repoName)
	}
}
