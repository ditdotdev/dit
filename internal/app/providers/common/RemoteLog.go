package common

import (
	"fmt"
	client "github.com/datadatdat/datadatdat-client-go"
	"os"
	"strconv"
)

func RemoteLog(repo string, remoteName string, tags []string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	remotes, _, _ := remotesApi.ListRemotes(ctx, repo).Execute()
	if len(remotes) == 0 {
		fmt.Println("remote is not set, run 'remote add' first")
		os.Exit(1)
	}
	first := true
	for _, r := range remotes {
		provider, err := ResolveProvider(r.Provider)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		gp, _ := provider.GetParameters(r.Properties)
		p := client.RemoteParameters{
			Provider:   r.Provider,
			Properties: gp,
		}
		commits, _, err := remotesApi.ListRemoteCommits(ctx, repo, r.Name).RemoteParameters(p).Tag(tags).Execute()
		if err == nil {
			for _, c := range commits {
				if !first {
					fmt.Println()
				} else {
					first = false
				}
				fmt.Println("Commit " + c.Id)
				ifContainsPrint(c.Properties, "author")
				ifContainsPrint(c.Properties, "user")
				ifContainsPrint(c.Properties, "email")
				ifContainsPrint(c.Properties, "timestamp")
				remoteTags, ok := c.Properties["tags"].(map[string]interface{})
				if ok {
					fmt.Print("Tags: ")
					for t, v := range remoteTags {
						if len(v.(string)) > 0 {
							fmt.Printf("%v=%v ", t, v)
						} else {
							fmt.Printf("%v ", t)
						}
					}
					fmt.Println()
				}
				ifContainsPrint(c.Properties, "message")
			}
		} else {
			fmt.Println(r.Name + " has not been initialized.")
		}
	}
}
