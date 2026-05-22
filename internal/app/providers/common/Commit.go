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

	repoProps, resp, err := repositoriesApi.GetRepository(ctx, repo).Execute()
	if err != nil || (resp != nil && resp.StatusCode >= 400) {
		fmt.Printf("Error: repository '%s' not found\n", repo)
		osExit(1)
	}
	metadata := Metadata{}.Load(repoProps.Properties)
	repoStatus, _, _ := repositoriesApi.GetRepositoryStatus(ctx, repo).Execute()
	sourceCommit := repoStatus.GetSourceCommit()
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
	// Pre-fix the error + response were both discarded: `response, _, _ :=
	// commitsApi.CreateCommit(...).Execute()`. On any server-side failure
	// (auth, repo-status race, ZFS error) `response` was nil; the
	// subsequent `response.Id` dereference panicked. Now we surface the
	// server's ApiError.Message via the shared handleOperationError
	// pathway used by Pull/Push, and exit non-zero so wrapping scripts
	// can react.
	response, _, err := commitsApi.CreateCommit(ctx, repo).Commit(commit).Execute()
	if handleOperationError(err) {
		osExit(1)
	}
	fmt.Println("Commit " + response.Id)
}
