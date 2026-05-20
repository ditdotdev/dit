package common

import (
	"fmt"
	"os"
	"strconv"

	client "github.com/datadatdat/datadatdat-client-go"
)

func Log(repo string, tags []string, port int) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)

	first := true
	opts := client.ListCommitsOpts{Tag: &tags}
	commits, resp, err := commitsApi.ListCommits(ctx, repo, &opts)
	if err != nil {
		fmt.Printf("Error listing commits for %s: %v\n", repo, err)
		os.Exit(1)
	}
	if resp != nil && resp.StatusCode >= 400 {
		fmt.Printf("Error listing commits for %s: server returned %d\n", repo, resp.StatusCode)
		os.Exit(1)
	}

	for _, commit := range commits {
		if !first {
			fmt.Println("")
		} else {
			first = false
		}
		metadata := commit.Properties
		fmt.Println("commit " + commit.Id)
		ifContainsPrint(metadata, "author")
		ifContainsPrint(metadata, "user")
		ifContainsPrint(metadata, "email")
		ifContainsPrint(metadata, "timestamp")

		tags, ok := metadata["tags"].(map[string]interface{})
		if ok {
			fmt.Print("Tags: ")
			for t, v := range tags {
				if len(v.(string)) > 0 {
					fmt.Printf("%v=%v ", t, v)
				} else {
					fmt.Printf("%v ", t)
				}
			}
			fmt.Println()
		}

		if metadata["message"] != "" {
			out := fmt.Sprintf("\n%v", metadata["message"])
			fmt.Println(out)
		}
	}
}
