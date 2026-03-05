package local

import (
	"datadatdat/internal/app/clients"
	"fmt"
)

func Stop(repo string, port int) error {
	docker := clients.Docker("", port)
	if _, err := docker.Stop(repo); err != nil {
		fmt.Printf("Error stopping container %s: %v\n", repo, err)
		return err
	}
	fmt.Println(repo + " stopped")
	return nil
}
