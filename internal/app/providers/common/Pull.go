package common

import (
	util "datadatdat/internal/app/utils"
	"fmt"
	"net/http"
	"os"
	"strconv"

	datadatdatclient "github.com/datadatdat/datadatdat-client-go"
)

const (
	DefaultRemoteName = "origin"
)

func Pull(repoName string, guid string, remoteName string, tags []string, metadataOnly bool, port int) {
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
		os.Exit(1)
	}
	remote, _, err := remotesApi.GetRemote(ctx, repoName, name).Execute()
	if err != nil || remote.Provider == "" {
		fmt.Printf("remote '%s' not found for repository '%s', run 'd3 remote add' first\n", name, repoName)
		os.Exit(1)
	}
	commit := datadatdatclient.Commit{
		Id: "id",
	}
	provider, err := ResolveProvider(remote.Provider)
	if err != nil {
		fmt.Println(err)
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
		var resp *http.Response
		var c *datadatdatclient.Commit
		c, resp, err = remotesApi.GetRemoteCommit(ctx, repoName, remote.Name, guid).RemoteParameters(params).Execute()
		if err != nil {
			handleRemoteError(err, resp, "")
			os.Exit(1)
		}
		commit = *c
	} else {
		remoteCommits, resp, err := remotesApi.ListRemoteCommits(ctx, repoName, remote.Name).RemoteParameters(params).Tag(tags).Execute()
		if err != nil {
			handleRemoteError(err, resp, "")
			os.Exit(1)
		}
		if len(remoteCommits) == 0 {
			fmt.Println("no matching commits found in remote, unable to pull latest")
			os.Exit(1)
		}
		commit = remoteCommits[0]
	}
	if commit.Id == "" {
		fmt.Println("remote commit not found")
		os.Exit(1)
	}
	op, _, err := operationsApi.Pull(ctx, repoName, remote.Name, commit.Id).RemoteParameters(params).MetadataOnly(metadataOnly).Execute()
	if handleOperationError(err) {
		os.Exit(1)
	}

	monitor := util.OperationMonitor(repoName, *op)
	if !monitor.Monitor(port) {
		os.Exit(1)
	}
}
