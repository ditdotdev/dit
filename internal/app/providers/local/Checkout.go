package local

import (
	"fmt"
	"github.com/antihax/optional"
	titanclient "github.com/datadatdat/titan-client-go"
	"os"
	"strconv"
	"titan/internal/app/clients"
)

func Checkout(repo string, guid string, tags []string, port int, context string) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)
	docker := clients.Docker(context, port)

	var sourceCommit string
	if guid == "" {
		if len(tags) > 0 {
			o := optional.NewInterface(tags)
			opts := titanclient.ListCommitsOpts{Tag: o}
			commits, _, _ := commitsApi.ListCommits(ctx, repo, &opts)
			if len(commits) == 0 {
				fmt.Println("no matching commits found")
				os.Exit(1)
			}
			sourceCommit = commits[0].Id
		} else {
			status, _, _ := repositoriesApi.GetRepositoryStatus(ctx, repo)
			if status.SourceCommit == "" {
				fmt.Println("no commits present, run 'titan commit' first")
				os.Exit(1)
			}
			sourceCommit = status.SourceCommit
		}
	} else {
		if len(tags) > 0 {
			fmt.Println("tags and commit cannot both be specified")
			os.Exit(1)
		}
		sourceCommit = guid
	}
	fmt.Println("Stopping container " + repo)
	if _, err := docker.Stop(repo); err != nil {
		fmt.Printf("Warning: Failed to stop container %s: %v\n", repo, err)
	}
	fmt.Println("Checkout " + sourceCommit)
	if _, err := commitsApi.CheckoutCommit(ctx, repo, sourceCommit); err != nil {
		fmt.Printf("Error checking out commit %s: %v\n", sourceCommit, err)
		os.Exit(1)
	}
	fmt.Println("Starting container " + repo)
	if _, err := docker.Start(repo); err != nil {
		fmt.Printf("Error starting container %s: %v\n", repo, err)
		os.Exit(1)
	}
	fmt.Println(sourceCommit + " checked out")
}
