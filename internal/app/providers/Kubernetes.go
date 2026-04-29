package providers

import (
	"datadatdat/internal/app"
	cmn "datadatdat/internal/app/providers/common"
	k8s "datadatdat/internal/app/providers/kubernetes"
	"fmt"
	"os"
	"strings"
)

type kubernetes struct {
	contextName             string
	host                    string
	portNum                 int
	datadatdatServerVersion string
	dockerRegistryUrl       string
}

func (k kubernetes) GetType() string {
	return "kubernetes"
}

func (k kubernetes) GetName() string {
	return k.contextName
}

func (k kubernetes) GetPort() int {
	return k.portNum
}

func (k kubernetes) Abort(repo string) {
	cmn.Abort(repo, k.portNum)
}

func (k kubernetes) Checkout(repo string, guid string, tags []string) {
	k8s.Checkout(repo, guid, tags, k.portNum)
}

func (k kubernetes) Clone(uri string, repo string, commit string, params []string, arguments []string, disablePortMap bool, tags []string) {
	cb := cmn.CloneCallbacks{
		Run: func(image, repoName string, envs, args []string, disablePortMap, privileged bool) (string, error) {
			// k8s.Run prints its own progress and returns nothing; the empty
			// string keeps cmn.Clone from printing a second status line, and
			// nil-error means common.Clone treats it as success.
			k8s.Run(image, repoName, envs, args, disablePortMap, privileged, false, k.portNum, k.contextName)
			return "", nil
		},
		Checkout: func(repoName, commitId string) {
			k8s.Checkout(repoName, commitId, nil, k.portNum)
		},
		Remove: func(repoName string, force bool) {
			k8s.Remove(repoName, force, k.portNum)
		},
	}
	cmn.Clone(uri, repo, commit, params, arguments, disablePortMap, tags, k.portNum, k.contextName, cb)
}

func (k kubernetes) Commit(repo string, message string, tags []string) {
	cmn.Commit(repo, message, tags, strings.TrimSpace(user), strings.TrimSpace(email), k.portNum)
}

func (k kubernetes) Copy(repo string, driver string, source string, path string) {
	fmt.Println("cp is not supported in kubernetes context")
	os.Exit(0)
}

func (k kubernetes) Fork(uri string, org string, name string) {
	cmn.Fork(uri, org, name)
}

func (k kubernetes) Delete(repo string, commit string, tags []string) {
	if commit != "" {
		if len(tags) > 0 {
			cmn.DeleteTags(repo, commit, tags, k.portNum)
		} else {
			cmn.DeleteCommit(repo, commit, k.portNum)
		}
	} else {
		fmt.Println("No object found to delete.")
	}
}

func (k kubernetes) Install(properties []string, verbose bool) {
	k8s.Install(k.datadatdatServerVersion, k.dockerRegistryUrl, verbose, k.portNum, k.contextName)
}

func (k kubernetes) List(context string) {
	k8s.List(k.contextName, k.portNum)
}

func (k kubernetes) Log(repo string, tags []string) {
	cmn.Log(repo, tags, k.portNum)
}

func (k kubernetes) Migrate(repo string, name string) {
	fmt.Println("migrate is not supported in kubernetes context")
	os.Exit(0)
}

func (k kubernetes) Pull(repo string, commit string, remoteName string, tags []string, metadataOnly bool) {
	cmn.Pull(repo, commit, remoteName, tags, metadataOnly, k.portNum)
}

func (k kubernetes) Push(repo string, commit string, remoteName string, tags []string, metadataOnly bool) {
	cmn.Push(repo, commit, remoteName, tags, metadataOnly, k.portNum)
}

func (k kubernetes) RemoteAdd(repo string, uri string, remoteName string, params map[string]string) {
	cmn.RemoteAdd(repo, uri, remoteName, params, k.portNum)
}

func (k kubernetes) RemoteList(repo string) {
	cmn.RemoteList(repo, k.portNum)
}

func (k kubernetes) RemoteLog(repo string, remoteName string, tags []string) {
	cmn.RemoteLog(repo, remoteName, tags, k.portNum)
}

func (k kubernetes) RemoteRemove(repo string, remote string) {
	cmn.RemoteRemove(repo, remote, k.portNum)
}

func (k kubernetes) Remove(repo string, force bool) {
	k8s.Remove(repo, force, k.portNum)
}

func (k kubernetes) Run(image string, repo string, environments []string, arguments []string, disablePortMap bool, privileged bool) {
	k8s.Run(image, repo, environments, arguments, disablePortMap, privileged, true, k.portNum, k.contextName)
}

func (k kubernetes) Start(repo string) {
	k8s.Start(repo, k.portNum)
}

func (k kubernetes) Status(repo string) {
	k8s.Status(repo, k.portNum, k.contextName)
}

func (k kubernetes) Stop(repo string) {
	k8s.Stop(repo, k.portNum)
}

func (k kubernetes) Tag(repo string, commit string, tags []string) {
	cmn.TagCommit(repo, commit, tags, k.portNum)
}

func (k kubernetes) Uninstall(force bool, removeImage bool) {
	k8s.Uninstall(force, removeImage, k.contextName, k.portNum)
}

func (k kubernetes) Upgrade(force bool, version string, finalize bool, path string) {
	panic("implement me")
}

func Kubernetes(contextName string, host string, port int) Provider {
	return kubernetes{
		contextName:             contextName,
		host:                    host,
		portNum:                 port,
		datadatdatServerVersion: app.DatadatdatVersion,
		dockerRegistryUrl:       "datadatdat",
	}
}
