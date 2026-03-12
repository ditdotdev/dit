package common

import (
	"datadatdat/internal/app/clients"
	"datadatdat/internal/app/providers/local"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/antihax/optional"
	client "github.com/datadatdat/datadatdat-client-go"
	"github.com/datadatdat/remote-sdk-go/remote"
	"net/http"
)

func Clone(uri string, repo string, guid string, params []string, args []string, disablePortMap bool, tags []string, port int, context string) {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)
	docker := clients.Docker(context, port)

	var parsedUri, _ = url.Parse(uri) //TODO handle err
	var repoName string
	if repo == "" {
		var p = strings.Split(parsedUri.Path, "/")
		repoName = p[len(p)-1]
	} else {
		repoName = repo
	}
	var commitId string
	if guid == "" && parsedUri.Fragment != "" {
		commitId = parsedUri.Fragment
	} else {
		commitId = guid
	}
	repository := client.Repository{
		Name:       repoName,
		Properties: make(map[string]interface{}),
	}
	plainUri := parsedUri.Scheme + "://" + parsedUri.Host + parsedUri.Path
	serverUrl := parsedUri.Scheme + "://" + parsedUri.Host
	if len(parsedUri.Query()) > 0 {
		tag := parsedUri.Query().Get("tag")
		tags = append(tags, tag)
	}
	var err error
	cleanup := false

	_, _, err = repositoriesApi.CreateRepository(ctx, repository)

	if err == nil {
		cleanup = true
		RemoteAdd(repoName, plainUri, "", nil, port) //TODO fix params
		rm, _, _ := remotesApi.GetRemote(ctx, repoName, "origin")
		gp, _ := remote.Get(rm.Provider).GetParameters(rm.Properties)
		p := client.RemoteParameters{
			Provider:   rm.Provider,
			Properties: gp,
		}
		commit := client.Commit{
			Id:         "id",
			Properties: map[string]interface{}{"foo": "bar"},
		}
		if commitId == "" {
			optTags := optional.NewInterface(tags)
			commitsOpts := &client.ListRemoteCommitsOpts{Tag: optTags}
			remoteCommits, resp, err := remotesApi.ListRemoteCommits(ctx, repoName, rm.Name, p, commitsOpts)
			if err != nil {
				handleRemoteError(err, resp, serverUrl)
				removeRepo(repoName, port, context)
				return
			}
			if len(remoteCommits) == 0 {
				fmt.Println("unable to find any matching commits in remote repository")
				removeRepo(repoName, port, context)
				return
			}
			commit = client.Commit{
				Id:         remoteCommits[0].Id,
				Properties: remoteCommits[0].Properties,
			}
		} else {
			if len(tags) > 0 {
				fmt.Println("tags cannot be specified with commit ID")
			}
			c, resp, err := remotesApi.GetRemoteCommit(ctx, repoName, rm.Name, commitId, p)
			if err != nil {
				handleRemoteError(err, resp, serverUrl)
				removeRepo(repoName, port, context)
				return
			}
			commit = client.Commit{
				Id:         c.Id,
				Properties: c.Properties,
			}
		}
		// Unwrap double-wrapped properties if present. The remote server may return
		// the full API response {id, properties} as the properties map rather than
		// just the inner properties sub-map containing v2/metadata.
		props := commit.Properties
		if innerProps, ok := props["properties"]; ok {
			if innerMap, ok := innerProps.(map[string]interface{}); ok {
				if _, hasV2 := innerMap["v2"]; hasV2 {
					props = innerMap
				}
			}
		}
		metadata := Metadata{}.Load(props)
		// Construct image reference from image name and tag
		imageRef := metadata.image.Image + ":" + metadata.image.Tag
		_, err = docker.InspectImage(imageRef)
		if err != nil {
			_, err = docker.Pull(imageRef)
			if err != nil {
				fmt.Println("Unable to find image " + imageRef + " for " + metadata.image.Image)
				os.Exit(1)
			}
		}
		_, err = docker.Pull(imageRef)
		if err != nil {
			fmt.Printf("Failed to pull image %s: %v\n", imageRef, err)
			os.Exit(1)
		}
		var envs []string
		for _, v := range metadata.environment {
			envs = append(envs, fmt.Sprintf("%v", v))
		}
		privileged := metadata.GetPrivileged()
		// Use disablePortMap from metadata if available, otherwise use the command-line flag
		metadataDisablePortMap := metadata.GetDisablePortMap()
		finalDisablePortMap := disablePortMap || metadataDisablePortMap
		m, err := local.Run(imageRef, repoName, envs, args, finalDisablePortMap, privileged, false, port, context)
		if err != nil {
			fmt.Printf("failed to run container: %v\n", err)
		} else {
			fmt.Println(m)
			Pull(repoName, commit.Id, "", make([]string, 0), false, port)
			local.Checkout(repoName, commit.Id, nil, port, context)
			cleanup = false
		}
	}
	if cleanup {
		removeRepo(repoName, port, context)
	}
}

func handleRemoteError(err error, resp *http.Response, uri string) {
	if resp != nil && resp.StatusCode == http.StatusUnauthorized {
		if uri != "" {
			fmt.Printf("authentication required: run 'd3 auth login --server %s --api-key <KEY>' to authenticate\n", uri)
		} else {
			fmt.Println("authentication required: run 'd3 auth login --api-key <KEY>' to authenticate")
		}
	} else {
		fmt.Printf("error communicating with remote: %v\n", err)
	}
}

func removeRepo(repoName string, port int, context string) {
	local.Remove(repoName, true, port, context)
	os.Exit(1)
}
