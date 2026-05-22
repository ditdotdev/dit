package common

import (
	"fmt"
	"strconv"

	client "github.com/datadatdat/datadatdat-client-go"
	_ "github.com/datadatdat/datadatdat-remote-go/datadatdat"

	// TODO: Uncomment when these providers have their Go packages implemented
	// _ "github.com/datadatdat/nop-remote-go/nop"
	"github.com/datadatdat/remote-sdk-go/remote"
	// _ "github.com/datadatdat/s3-remote-go/s3"
	// _ "github.com/datadatdat/s3web-remote-go/s3web"
	// _ "github.com/datadatdat/ssh-remote-go/ssh"
)

func RemoteAdd(repo string, uri string, remoteName string, params map[string]string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	var name string
	if remoteName != "" {
		name = remoteName
	} else {
		name = "origin"
	}
	_, _, err := remotesApi.GetRemote(ctx, repo, name).Execute()
	if err == nil {
		fmt.Println("remote " + name + " already exists for " + repo)
		osExit(1)
	}
	parsed, err := remote.ParseURL(uri, params)
	if err != nil {
		fmt.Printf("Error parsing URI '%s': %v\n", uri, err)
		osExit(1)
	}
	r := client.Remote{
		Provider:   parsed.Provider,
		Name:       name,
		Properties: parsed.Properties,
	}
	_, _, err = remotesApi.CreateRemote(ctx, repo).Remote(r).Execute()
	if err != nil {
		fmt.Printf("Error creating remote: %v\n", err)
		osExit(1)
	}
	m, _, _ := repositoriesApi.GetRepository(ctx, repo).Execute()
	metadata := m.Properties
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["remote"] = name
	newRepo := client.Repository{
		Name:       repo,
		Properties: metadata,
	}
	_, _, _ = repositoriesApi.UpdateRepository(ctx, repo).Repository(newRepo).Execute()
}
