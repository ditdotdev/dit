package local

import (
	"datadatdat/internal/app/clients"
	"fmt"
)

func Start(repo string, port int) {
	docker := clients.Docker("", port)
	if _, err := docker.Start(repo); err != nil {
		fmt.Printf("Error starting container %s: %v\n", repo, err)
		return
	}
	fmt.Println(repo + " started")
}
