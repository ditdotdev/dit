package common

import (
	"fmt"
	"strconv"

	client "github.com/ditdotdev/dit-client-go"
	// Blank-import every remote provider so its init() registers with the SDK
	// registry. Since the remote-sdk-go registry refactor, remote.ParseURL
	// resolves a URI by asking each *registered* provider (via FromURL), so a
	// provider that isn't imported here can't be parsed — `dit remote add
	// ssh://...` would fail with "no remote provider found". The old scheme-based
	// ParseURL didn't need this, which is why these were previously commented out.
	_ "github.com/ditdotdev/dit-remote-go/dit"
	_ "github.com/ditdotdev/nop-remote-go/nop"
	_ "github.com/ditdotdev/s3-remote-go/s3"
	_ "github.com/ditdotdev/s3web-remote-go/s3web"
	_ "github.com/ditdotdev/ssh-remote-go/ssh"

	"github.com/ditdotdev/remote-sdk-go/remote"
)

func RemoteAdd(repo string, uri string, remoteName string, params map[string]string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	var name string
	if remoteName != "" {
		name = remoteName
	} else {
		name = DefaultRemoteName
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
