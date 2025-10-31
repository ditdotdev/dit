package common

import (
	"fmt"
	"strconv"
	"strings"

	client "github.com/datadatdat/datadatdat-client-go"
)

func DeleteCommit(repo string, commit string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	if _, err := commitsApi.DeleteCommit(ctx, repo, commit).Execute(); err != nil {
		fmt.Printf("Error deleting commit %s: %v\n", commit, err)
		return
	}
	fmt.Println(commit + " deleted")
}

func DeleteTags(repo string, commit string, tags []string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	c, _, _ := commitsApi.GetCommit(ctx, repo, commit).Execute()
	cTags := c.Properties["tags"].(map[string]string)
	for _, t := range tags {
		if strings.Contains(t, "=") {
			s := strings.Split(t, "=")
			k := s[0]
			v := s[1]

			val, ok := cTags[k]
			if ok && val == v {
				delete(cTags, k)
			}
		} else {
			delete(cTags, t)
		}
	}
	metadata := Metadata{}.Load(c.Properties)
	metadata.SetTags(cTags)
	cm := client.Commit{
		Id:         c.Id,
		Properties: metadata.ToMap(),
	}
	if _, _, err := commitsApi.UpdateCommit(ctx, repo, c.Id).Commit(cm).Execute(); err != nil {
		fmt.Printf("Error updating commit tags: %v\n", err)
	}
}
