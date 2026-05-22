package providers

import (
	"datadatdat/internal/app"
	cmn "datadatdat/internal/app/providers/common"
	lcl "datadatdat/internal/app/providers/local"
	"datadatdat/internal/app/utils"
	"fmt"
	"os"
	"strings"
)

var ce = utils.CommandExecutor(60, false)

// gitIdentity returns the git config user.name + user.email, looked up
// lazily on first use. Previously these were package-level vars that
// forked `git` at package init for EVERY CLI invocation — including
// pure-read commands like `d3 ls`, `d3 status`, `d3 log` — costing
// 50-200ms of fork+exec overhead and failing outright in containers
// without git installed. Only Commit and Migrate actually use these
// values; defer the lookup until then.
//
// Both kubernetes/ and local/ Commit/Migrate paths route through here.
func gitIdentity() (string, string) {
	u, _ := ce.Exec("git", "config", "user.name")
	e, _ := ce.Exec("git", "config", "user.email")
	return strings.TrimSpace(u), strings.TrimSpace(e)
}

type local struct {
	contextName             string
	host                    string
	portNum                 int
	datadatdatServerVersion string
	dockerRegistryUrl       string
}

func (l local) GetType() string {
	return "docker"
}

func (l local) GetName() string {
	return l.contextName
}

func (l local) GetPort() int {
	return l.portNum
}

func (l local) Abort(repo string) {
	cmn.Abort(repo, l.portNum)
}

func (l local) Checkout(repo string, guid string, tags []string) {
	lcl.Checkout(repo, guid, tags, l.portNum, l.contextName)
}

func (l local) Clone(uri string, repo string, commit string, params []string, arguments []string, disablePortMap bool, tags []string) {
	cb := cmn.CloneCallbacks{
		Run: func(image, repoName string, envs, args []string, disablePortMap, privileged bool) (string, error) {
			return lcl.Run(image, repoName, envs, args, disablePortMap, privileged, false, l.portNum, l.contextName)
		},
		Checkout: func(repoName, commitId string) {
			lcl.Checkout(repoName, commitId, nil, l.portNum, l.contextName)
		},
		Remove: func(repoName string, force bool) {
			lcl.Remove(repoName, force, l.portNum, l.contextName)
		},
	}
	cmn.Clone(uri, repo, commit, params, arguments, disablePortMap, tags, l.portNum, l.contextName, cb)
}

func (l local) Commit(repo string, message string, tags []string) {
	u, e := gitIdentity()
	cmn.Commit(repo, message, tags, u, e, l.portNum)
}

func (l local) Copy(repo string, driver string, source string, path string) {
	lcl.Copy(repo, driver, source, path, l.portNum, l.contextName)
}

func (l local) Fork(uri string, org string, name string) {
	cmn.Fork(uri, org, name)
}

func (l local) Delete(repo string, commit string, tags []string) {
	if commit != "" {
		if len(tags) > 0 {
			cmn.DeleteTags(repo, commit, tags, l.portNum)
		} else {
			cmn.DeleteCommit(repo, commit, l.portNum)
		}
	} else {
		fmt.Println("No object found to delete.")
	}
}

func (l local) Install(properties []string, verbose bool) {
	registry := l.dockerRegistryUrl // default
	// Parse properties to override registry if specified
	for _, prop := range properties {
		if strings.HasPrefix(prop, "registry=") {
			registry = strings.TrimPrefix(prop, "registry=")
			break
		}
	}
	lcl.Install(l.datadatdatServerVersion, registry, verbose, l.portNum, l.contextName)
}

func (l local) List(context string) {
	lcl.List(context, l.portNum)
}

func (l local) Log(repo string, tags []string) {
	cmn.Log(repo, tags, l.portNum)
}

func (l local) Migrate(repo string, name string) {
	u, e := gitIdentity()
	lcl.Migrate(repo, name, u, e, cmn.Commit, l.portNum, l.contextName)
}

func (l local) Pull(repo string, commit string, remoteName string, tags []string, metadataOnly bool) {
	cmn.Pull(repo, commit, remoteName, tags, metadataOnly, l.portNum)
}

func (l local) Push(repo string, commit string, remoteName string, tags []string, metadataOnly bool) {
	cmn.Push(repo, commit, remoteName, tags, metadataOnly, l.portNum)
}

func (l local) RemoteAdd(repo string, uri string, remoteName string, params map[string]string) {
	cmn.RemoteAdd(repo, uri, remoteName, params, l.portNum)
}

func (l local) RemoteList(repo string) {
	cmn.RemoteList(repo, l.portNum)
}

func (l local) RemoteLog(repo string, remoteName string, tags []string) {
	cmn.RemoteLog(repo, remoteName, tags, l.portNum)
}

func (l local) RemoteRemove(repo string, remote string) {
	cmn.RemoteRemove(repo, remote, l.portNum)
}

func (l local) Remove(repo string, force bool) {
	lcl.Remove(repo, force, l.portNum, l.contextName)
}

func (l local) Run(image string, repo string, environments []string, arguments []string, disablePortMap bool, privileged bool) {
	s, err := lcl.Run(image, repo, environments, arguments, disablePortMap, privileged, true, l.portNum, l.contextName)
	fmt.Println(s)
	if err != nil {
		os.Exit(1)
	}
}

func (l local) Start(repo string) {
	if err := lcl.Start(repo, l.portNum); err != nil {
		os.Exit(1)
	}
}

func (l local) Status(repo string) {
	cmn.Status(repo, l.portNum, l.contextName)
}

func (l local) Stop(repo string) {
	if err := lcl.Stop(repo, l.portNum); err != nil {
		os.Exit(1)
	}
}

func (l local) Tag(repo string, commit string, tags []string) {
	cmn.TagCommit(repo, commit, tags, l.portNum)
}

func (l local) Uninstall(force bool, removeImage bool) {
	lcl.Uninstall(l.datadatdatServerVersion, force, removeImage, l.portNum, l.contextName)
}

func (l local) Upgrade(force bool, version string, finalize bool, path string) {
	targetVersion := l.datadatdatServerVersion
	if version != "" {
		targetVersion = version
	}
	fmt.Println("Upgrading datadatdat infrastructure to " + targetVersion)
	lcl.Install(targetVersion, l.dockerRegistryUrl, false, l.portNum, l.contextName)
}

func Local(contextName string, host string, port int) Provider {
	return local{
		contextName:             contextName,
		host:                    host,
		portNum:                 port,
		datadatdatServerVersion: app.DatadatdatVersion,
		dockerRegistryUrl:       "datadatdat",
	}
}
