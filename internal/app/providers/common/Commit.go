package common

import (
	"fmt"
	"strconv"
	"strings"

	client "github.com/datadatdat/datadatdat-client-go"
	"github.com/google/uuid"
)

func Commit(repo string, message string, tags []string, user string, email string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	repoProps, _, _ := repositoriesApi.GetRepository(ctx, repo).Execute()
	metadata := Metadata{}.Load(repoProps.Properties)
	repoStatus, _, _ := repositoriesApi.GetRepositoryStatus(ctx, repo).Execute() //TODO handle this error
	var sourceCommit string
	if repoStatus.SourceCommit != nil {
		sourceCommit = *repoStatus.SourceCommit
	}
	tagMetadata := make(map[string]string)
	for _, tag := range tags {
		var k string
		var v string
		if strings.Contains(tag, "=") {
			s := strings.Split(tag, "=")
			k = s[0]
			v = s[1]
		} else {
			k = tag
			v = ""
		}
		tagMetadata[k] = v
	}
	metadata.SetEmail(email)
	metadata.SetUser(user)
	metadata.SetMessage(message)
	metadata.SetTags(tagMetadata)
	metadata.SetSource(sourceCommit)

	guid := uuid.New().String()
	guid = strings.ReplaceAll(guid, "-", "")
	commit := client.Commit{
		Id:         guid,
		Properties: metadata.ToMap(),
	}
	response, _, _ := commitsApi.CreateCommit(ctx, repo).Commit(commit).Execute()
	fmt.Println("Commit " + response.Id)
}
