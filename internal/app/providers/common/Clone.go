package common

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	client "github.com/datadatdat/datadatdat-client-go"
	"net/http"
)

// CloneCallbacks lets callers in providers/{local,kubernetes} hand
// Clone the right Run/Checkout/Remove implementation for their context.
//
// Pre-fix the package directly imported and called local.Run / local.Checkout /
// local.Remove regardless of which provider's Clone() invoked common.Clone —
// so `d3 clone --context k8s ...` was silently routed through the docker code
// path, surfacing as "Creating docker volume hello-clone-s3web_v0" mid-run on
// kubernetes contexts and failing immediately. Surfaced by
// kubernetes-remote-tests.bats tests 9-10. Routing through caller-supplied
// callbacks keeps Clone's metadata/remote-fetch logic provider-agnostic
// while letting each provider plug in its own runtime.
type CloneCallbacks struct {
	// Run brings up a container/pod for `repoName` from `image`, returning a
	// human-readable status string (printed to stdout on success). Error
	// triggers Remove + os.Exit via removeRepo.
	Run func(image, repoName string, envs, args []string, disablePortMap, privileged bool) (string, error)
	// Checkout switches `repoName` to the given commit. Errors here are
	// already printed by the underlying call; this fn doesn't propagate.
	Checkout func(repoName, commitId string)
	// Remove tears down `repoName` (used both at end of a failed Clone and
	// when Clone bails early after a remote error).
	Remove func(repoName string, force bool)
}

func Clone(uri string, repo string, guid string, params []string, args []string, disablePortMap bool, tags []string, port int, context string, cb CloneCallbacks) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := newDocker(context, port)

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
	_, resp, err := repositoriesApi.CreateRepository(ctx).Repository(repository).Execute()
	if err != nil {
		handleRemoteError(err, resp, "")
		osExit(1)
	}

	RemoteAdd(repoName, plainUri, "", nil, port) //TODO fix params
	rm, _, _ := remotesApi.GetRemote(ctx, repoName, "origin").Execute()
	provider, err := ResolveProvider(rm.Provider)
	if err != nil {
		fmt.Println(err)
		osExit(1)
	}
	gp, _ := provider.GetParameters(rm.Properties)
	p := client.RemoteParameters{
		Provider:   rm.Provider,
		Properties: gp,
	}
	commit := client.Commit{
		Id:         "id",
		Properties: map[string]interface{}{"foo": "bar"},
	}
	if commitId == "" {
		remoteCommits, resp, err := remotesApi.ListRemoteCommits(ctx, repoName, rm.Name).RemoteParameters(p).Tag(tags).Execute()
		if err != nil {
			handleRemoteError(err, resp, serverUrl)
			removeRepo(repoName, cb)
			return
		}
		if len(remoteCommits) == 0 {
			fmt.Println("unable to find any matching commits in remote repository")
			removeRepo(repoName, cb)
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
		c, resp, err := remotesApi.GetRemoteCommit(ctx, repoName, rm.Name, commitId).RemoteParameters(p).Execute()
		if err != nil {
			handleRemoteError(err, resp, serverUrl)
			removeRepo(repoName, cb)
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
			osExit(1)
		}
	}
	_, err = docker.Pull(imageRef)
	if err != nil {
		fmt.Printf("Failed to pull image %s: %v\n", imageRef, err)
		osExit(1)
	}
	var envs []string
	for _, v := range metadata.environment {
		envs = append(envs, fmt.Sprintf("%v", v))
	}
	privileged := metadata.GetPrivileged()
	// Use disablePortMap from metadata if available, otherwise use the command-line flag
	metadataDisablePortMap := metadata.GetDisablePortMap()
	finalDisablePortMap := disablePortMap || metadataDisablePortMap
	m, err := cb.Run(imageRef, repoName, envs, args, finalDisablePortMap, privileged)
	if err != nil {
		fmt.Printf("failed to run container: %v\n", err)
		removeRepo(repoName, cb)
		return
	}
	if m != "" {
		// kubernetes Run prints its own progress to stdout and returns ""
		// here; only print non-empty status from the callback.
		fmt.Println(m)
	}
	Pull(repoName, commit.Id, "", make([]string, 0), false, port)
	cb.Checkout(repoName, commit.Id)
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

func removeRepo(repoName string, cb CloneCallbacks) {
	cb.Remove(repoName, true)
	osExit(1)
}
