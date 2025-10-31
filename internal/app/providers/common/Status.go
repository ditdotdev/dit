package common

import (
	"datadatdat/internal/app/clients"
	"fmt"
	"strconv"
)

// TODO pass this from provider as param
func getContainersStatus(port int, context string) []runtimeStatus {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := clients.Docker(context, port)

	repos, _, _ := repositoriesApi.ListRepositories(ctx).Execute()
	var r []runtimeStatus
	for _, repo := range repos {
		status, err := docker.GetValFromContainer(repo.Name, "State", "Status")
		if err == nil {
			r = append(r, RuntimeStatus(repo.Name, status))
		} else {
			r = append(r, RuntimeStatus(repo.Name, "detached"))
		}
	}
	return r
}

/*
*
https://programming.guide/go/formatting-byte-size-to-human-readable-format.html
*/
func ByteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func Status(repo string, port int, context string) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	s, _, _ := repositoriesApi.GetRepositoryStatus(ctx, repo).Execute()
	for _, r := range getContainersStatus(port, context) {
		if r.name == repo {
			o := fmt.Sprintf("%20s %s", "Status: ", r.status)
			fmt.Println(o)
		}
	}
	if s.LastCommit != nil && *s.LastCommit != "" {
		o := fmt.Sprintf("%20s %s", "Last Commit: ", *s.LastCommit)
		fmt.Println(o)
	}
	if s.SourceCommit != nil && *s.SourceCommit != "" {
		o := fmt.Sprintf("%20s %s", "Source Commit: ", *s.SourceCommit)
		fmt.Println(o)
	}
	vols, _, _ := volumesApi.ListVolumes(ctx, repo).Execute()
	o := fmt.Sprintf("%-30s  %-12s  %s", "Volume", "Uncompressed", "Compressed")
	fmt.Println(o)
	for _, v := range vols {
		vstat, _, _ := volumesApi.GetVolumeStatus(ctx, repo, v.Name).Execute()
		o := fmt.Sprintf("%-30s  %-12s  %s", vstat.Properties["path"],
			ByteCountBinary(vstat.LogicalSize),
			ByteCountBinary(vstat.ActualSize),
		)
		fmt.Println(o)
	}
}
