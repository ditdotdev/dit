package local

import (
	"fmt"
	"datadatdat/internal/app/clients"
)

func Stop(repo string, port int) {
	docker := clients.Docker("", port)
	if _, err := docker.Stop(repo); err != nil {
		fmt.Printf("Error stopping container %s: %v\n", repo, err)
		return
	}
	fmt.Println(repo + " stopped")
}
