package local

import (
	"fmt"
	"strconv"
)

func List(context string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)
	docker := newDocker("", port)

	repos, _, _ := repositoriesApi.ListRepositories(ctx).Execute()
	for _, repo := range repos {
		var status string
		info, err := docker.GetValFromContainer(repo.Name, "State", "Status")
		if err == nil {
			status = info
		} else {
			status = "detached"
		}
		l := fmt.Sprintf("%-12s  %-20s  %s", context, repo.Name, status)
		fmt.Println(l)
	}
}
