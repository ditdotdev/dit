package local

import (
	"fmt"
	"strconv"
)

func Checkout(repo string, guid string, tags []string, port int, context string) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := newDocker(context, port)

	var sourceCommit string
	if guid == "" {
		if len(tags) > 0 {
			commits, _, _ := commitsApi.ListCommits(ctx, repo).Tag(tags).Execute()
			if len(commits) == 0 {
				fmt.Println("no matching commits found")
				osExit(1)
			}
			sourceCommit = commits[0].Id
		} else {
			status, _, _ := repositoriesApi.GetRepositoryStatus(ctx, repo).Execute()
			if status.GetSourceCommit() == "" {
				fmt.Println("no commits present, run 'd3 commit' first")
				osExit(1)
			}
			sourceCommit = status.GetSourceCommit()
		}
	} else {
		if len(tags) > 0 {
			fmt.Println("tags and commit cannot both be specified")
			osExit(1)
		}
		sourceCommit = guid
	}
	fmt.Println("Stopping container " + repo)
	if _, err := docker.Stop(repo); err != nil {
		fmt.Printf("Warning: Failed to stop container %s: %v\n", repo, err)
	}
	fmt.Println("Checkout " + sourceCommit)
	if _, err := commitsApi.CheckoutCommit(ctx, repo, sourceCommit).Execute(); err != nil {
		fmt.Printf("Error checking out commit %s: %v\n", sourceCommit, err)
		osExit(1)
	}
	fmt.Println("Starting container " + repo)
	if _, err := docker.Start(repo); err != nil {
		fmt.Printf("Error starting container %s: %v\n", repo, err)
		osExit(1)
	}
	fmt.Println(sourceCommit + " checked out")
}
