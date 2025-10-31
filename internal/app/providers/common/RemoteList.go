package common

import (
	"fmt"
	"strconv"

	"github.com/datadatdat/remote-sdk-go/remote"
)

func RemoteList(repo string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	remotes, _, _ := remotesApi.ListRemotes(ctx, repo).Execute()
	fmt.Printf("%-20s %-20s\n", "REMOTE", "URI") //TODO get proper os line separator
	for _, r := range remotes {
		url, _, _ := remote.Get(r.Provider).ToURL(r.Properties)
		fmt.Printf("%-20s %-20s\n", r.Name, url)
	}
}
