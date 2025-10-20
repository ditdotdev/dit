package common

import (
	"fmt"
	"os"
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
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)

	var name string
	if remoteName != "" {
		name = remoteName
	} else {
		name = "origin"
	}
	_, _, err := remotesApi.GetRemote(ctx, repo, name)
	if err == nil {
		fmt.Println("remote " + name + " already exists for " + repo)
		os.Exit(1)
	}
	provider, props, _, _, err := remote.ParseURL(uri, params)
	if err != nil {
		fmt.Printf("Error parsing URI '%s': %v\n", uri, err)
		os.Exit(1)
	}
	fmt.Printf("DEBUG: ParseURL returned provider='%s', props=%+v\n", provider, props)
	r := client.Remote{
		Provider:   provider,
		Name:       name,
		Properties: props,
	}
	fmt.Printf("DEBUG: Creating remote with provider='%s', name='%s'\n", r.Provider, r.Name)
	_, _, err = remotesApi.CreateRemote(ctx, repo, r)
	if err != nil {
		fmt.Printf("Error creating remote: %v\n", err)
		os.Exit(1)
	}
	m, _, _ := repositoriesApi.GetRepository(ctx, repo)
	metadata := m.Properties
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["remote"] = name
	newRepo := client.Repository{
		Name:       repo,
		Properties: metadata,
	}
	_, _, _ = repositoriesApi.UpdateRepository(ctx, repo, newRepo)
}
