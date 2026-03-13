package common

import (
	util "datadatdat/internal/app/utils"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/antihax/optional"
	datadatdatclient "github.com/datadatdat/datadatdat-client-go"
	rm "github.com/datadatdat/remote-sdk-go/remote"
)

func ensureRemoteRepoExists(properties map[string]interface{}) error {
	apiBaseURL, _ := properties["api_base_url"].(string)
	org, _ := properties["org"].(string)
	repo, _ := properties["repo"].(string)
	if apiBaseURL == "" || org == "" || repo == "" {
		return fmt.Errorf("missing api_base_url, org, or repo in remote properties")
	}
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s", apiBaseURL, org, repo)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	if apiKey := os.Getenv("DATADATDAT_API_KEY"); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned %d when creating remote repository %s/%s", resp.StatusCode, org, repo)
	}
	return nil
}

func Push(repoName string, guid string, remoteName string, tags []string, metadataOnly bool, port int) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)

	var name string
	if remoteName == "" {
		name = "origin"
	} else {
		name = remoteName
	}
	_, _, err := remotesApi.ListRemotes(ctx, repoName)
	if err != nil {
		fmt.Println("remote is not set, run 'remote add' first")
		os.Exit(1)
	}
	repoStatus, _, _ := repositoriesApi.GetRepositoryStatus(ctx, repoName)
	if repoStatus.LastCommit == "" {
		fmt.Println("container has no history, run 'commit' to first commit state")
		os.Exit(1)
	}
	remote, _, err := remotesApi.GetRemote(ctx, repoName, name)
	if err != nil || remote.Provider == "" {
		fmt.Printf("remote '%s' not found for repository '%s', run 'd3 remote add' first\n", name, repoName)
		os.Exit(1)
	}
	commit := datadatdatclient.Commit{
		Id: "id",
	}
	if remote.Provider == "datadatdat" {
		if err := ensureRemoteRepoExists(remote.Properties); err != nil {
			fmt.Printf("failed to create remote repository: %s\n", err)
			os.Exit(1)
		}
	}
	provider := rm.Get(remote.Provider)
	if provider == nil {
		fmt.Printf("unknown remote provider '%s'\n", remote.Provider)
		os.Exit(1)
	}
	p, _ := provider.GetParameters(remote.Properties)
	params := datadatdatclient.RemoteParameters{
		Provider:   remote.Provider,
		Properties: p,
	}
	if guid != "" {
		if len(tags) > 0 {
			fmt.Println("tags cannot be specified when commit is also specified")
			os.Exit(1)
		}
		commit, _, _ = commitsApi.GetCommit(ctx, repoName, guid)
	} else {
		if len(tags) == 0 {
			commit, _, _ = commitsApi.GetCommit(ctx, repoName, repoStatus.LastCommit)
		} else {
			optTags := optional.NewInterface(tags)
			commitsOpts := &datadatdatclient.ListCommitsOpts{Tag: optTags}
			commits, _, _ := commitsApi.ListCommits(ctx, repoName, commitsOpts)
			if len(commits) == 0 {
				fmt.Println("no matching commits found, unable to push latest")
				os.Exit(1)
			}
			commit = commits[0]
		}
	}
	if commit.Id == "" {
		fmt.Println("no matching commits found, unable to push latest")
		os.Exit(1)
	}
	pushOpts := &datadatdatclient.PushOpts{
		MetadataOnly: optional.NewBool(metadataOnly),
	}
	op, _, err := operationsApi.Push(ctx, repoName, remote.Name, commit.Id, params, pushOpts)
	if err != nil {
		if e, ok := err.(datadatdatclient.GenericOpenAPIError); ok {
			m := e.Model().(datadatdatclient.ApiError)
			fmt.Println(m.Message)
			os.Exit(1)
		}
	}
	monitor := util.OperationMonitor(repoName, op)
	if !monitor.Monitor(port) {
		os.Exit(1)
	}
}
