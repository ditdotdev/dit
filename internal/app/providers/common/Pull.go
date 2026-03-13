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

const (
	DefaultRemoteName = "origin"
)

func Pull(repoName string, guid string, remoteName string, tags []string, metadataOnly bool, port int) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)

	var name string
	if remoteName == "" {
		name = DefaultRemoteName
	} else {
		name = remoteName
	}
	_, _, err := remotesApi.ListRemotes(ctx, repoName)
	if err != nil {
		fmt.Println("remote is not set, run 'remote add' first")
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
		var resp *http.Response
		commit, resp, err = remotesApi.GetRemoteCommit(ctx, repoName, remote.Name, guid, params)
		if err != nil {
			handleRemoteError(err, resp, "")
			os.Exit(1)
		}
	} else {
		o := optional.NewInterface(tags)
		opts := datadatdatclient.ListRemoteCommitsOpts{Tag: o}
		remoteCommits, resp, err := remotesApi.ListRemoteCommits(ctx, repoName, remote.Name, params, &opts)
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
	pullOpts := &datadatdatclient.PullOpts{
		MetadataOnly: optional.NewBool(metadataOnly),
	}
	op, _, _ := operationsApi.Pull(ctx, repoName, remote.Name, commit.Id, params, pullOpts)

	monitor := util.OperationMonitor(repoName, op)
	if !monitor.Monitor(port) {
		os.Exit(1)
	}
}
