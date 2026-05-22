package common

import (
	"fmt"
	"strconv"
)

func RemoteList(repo string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	remotes, _, _ := remotesApi.ListRemotes(ctx, repo).Execute()
	fmt.Printf("%-20s %-20s\n", "REMOTE", "URI") //TODO get proper os line separator
	for _, r := range remotes {
		provider, err := ResolveProvider(r.Provider)
		if err != nil {
			fmt.Println(err)
			osExit(1)
		}
		url, _, _ := provider.ToURL(r.Properties)
		fmt.Printf("%-20s %-20s\n", r.Name, url)
	}
}
