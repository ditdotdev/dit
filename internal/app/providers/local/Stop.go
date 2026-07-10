// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"fmt"
)

func Stop(repo string, port int) error {
	docker := newDocker("", port)
	if _, err := docker.Stop(repo); err != nil {
		fmt.Printf("Error stopping container %s: %v\n", repo, err)
		return err
	}
	fmt.Println(repo + " stopped")
	return nil
}
