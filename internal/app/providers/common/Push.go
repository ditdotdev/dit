package common

import (
	util "github.com/ditdotdev/dit/internal/app/utils"
	"fmt"
	"net/http"
	"os"
	"strconv"

	ditclient "github.com/ditdotdev/dit-client-go"
)

// providerDit is the remote provider name for the dit-hosted backend.
const providerDit = "dit"

func ensureRemoteRepoExists(properties map[string]interface{}) error {
	apiBaseURL, _ := properties["api_base_url"].(string)
	org, _ := properties["org"].(string)
	repo, _ := properties["repo"].(string)
	if apiBaseURL == "" || org == "" || repo == "" {
		return fmt.Errorf("missing api_base_url, org, or repo in remote properties")
	}
	// Allow overriding the host-visible gateway URL. Needed in environments where
	// the remote URL uses a Docker-internal hostname (e.g. dit-api-gateway)
	// that isn't resolvable from the host process running dit.
	if hostGateway := os.Getenv("DIT_HOST_GATEWAY"); hostGateway != "" {
		apiBaseURL = hostGateway
	}
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s", apiBaseURL, org, repo)

	// Check if repo already exists before creating. Creating an existing repo
	// overwrites its manifest with an empty one, destroying commit history.
	if getReq, err := http.NewRequest(http.MethodGet, url, nil); err == nil { // #nosec G704 -- URL built from configured remote properties, not untrusted input
		if apiKey := os.Getenv("DIT_API_KEY"); apiKey != "" {
			getReq.Header.Set("Authorization", "Bearer "+apiKey)
		}
		if getResp, err := http.DefaultClient.Do(getReq); err == nil { // #nosec G704 -- request uses validated URL from remote config
			_ = getResp.Body.Close()
			if getResp.StatusCode == http.StatusOK {
				return nil // Repo already exists
			}
		}
	}

	req, err := http.NewRequest(http.MethodPost, url, nil) // #nosec G704 -- URL built from configured remote properties, not untrusted input
	if err != nil {
		return err
	}
	if apiKey := os.Getenv("DIT_API_KEY"); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req) // #nosec G704 -- URL from configured remote properties
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned %d when creating remote repository %s/%s", resp.StatusCode, org, repo)
	}
	return nil
}

func Push(repoName string, guid string, remoteName string, tags []string, metadataOnly bool, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	var name string
	if remoteName == "" {
		name = DefaultRemoteName
	} else {
		name = remoteName
	}
	_, _, err := remotesApi.ListRemotes(ctx, repoName).Execute()
	if err != nil {
		fmt.Println("remote is not set, run 'remote add' first")
		osExit(1)
	}
	repoStatus, _, _ := repositoriesApi.GetRepositoryStatus(ctx, repoName).Execute()
	if repoStatus.GetLastCommit() == "" {
		fmt.Println("container has no history, run 'commit' to first commit state")
		osExit(1)
	}
	remote, _, err := remotesApi.GetRemote(ctx, repoName, name).Execute()
	if err != nil || remote.Provider == "" {
		fmt.Printf("remote '%s' not found for repository '%s', run 'dit remote add' first\n", name, repoName)
		osExit(1)
	}
	commit := ditclient.Commit{
		Id: "id",
	}
	if remote.Provider == providerDit {
		if err := ensureRemoteRepoExists(remote.Properties); err != nil {
			fmt.Printf("failed to create remote repository: %s\n", err)
			osExit(1)
		}
	}
	provider, err := ResolveProvider(remote.Provider)
	if err != nil {
		fmt.Println(err)
		osExit(1)
	}
	p, _ := provider.GetParameters(remote.Properties)
	params := ditclient.RemoteParameters{
		Provider:   remote.Provider,
		Properties: p,
	}
	if guid != "" {
		if len(tags) > 0 {
			fmt.Println("tags cannot be specified when commit is also specified")
			osExit(1)
		}
		c, _, _ := commitsApi.GetCommit(ctx, repoName, guid).Execute()
		if c != nil {
			commit = *c
		}
	} else {
		if len(tags) == 0 {
			c, _, _ := commitsApi.GetCommit(ctx, repoName, repoStatus.GetLastCommit()).Execute()
			if c != nil {
				commit = *c
			}
		} else {
			commits, _, _ := commitsApi.ListCommits(ctx, repoName).Tag(tags).Execute()
			if len(commits) == 0 {
				fmt.Println("no matching commits found, unable to push latest")
				osExit(1)
			}
			commit = commits[0]
		}
	}
	if commit.Id == "" {
		fmt.Println("no matching commits found, unable to push latest")
		osExit(1)
	}
	// Check if commit already exists in the remote to prevent duplicate pushes.
	// Skip for metadata-only pushes (--update-only), which re-push to sync tags.
	if !metadataOnly {
		remoteCommits, _, listErr := remotesApi.ListRemoteCommits(ctx, repoName, name).RemoteParameters(ditclient.RemoteParameters{
			Provider:   remote.Provider,
			Properties: p,
		}).Execute()
		if listErr == nil {
			for _, rc := range remoteCommits {
				if rc.Id == commit.Id {
					fmt.Printf("commit %s exists in remote '%s'\n", commit.Id, name)
					osExit(1)
				}
			}
		}
	}
	op, _, err := operationsApi.Push(ctx, repoName, remote.Name, commit.Id).RemoteParameters(params).MetadataOnly(metadataOnly).Execute()
	if handleOperationError(err) {
		osExit(1)
	}
	monitor := util.OperationMonitor(repoName, *op)
	if !monitor.Monitor(port) {
		osExit(1)
	}
}
