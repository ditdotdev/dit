package common

import (
	"fmt"
	"strconv"
)

func RemoteRemove(repo string, remote string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	if _, err := remotesApi.DeleteRemote(ctx, repo, remote).Execute(); err != nil {
		fmt.Printf("Error removing remote %s from %s: %v\n", remote, repo, err)
		return
	}
	fmt.Println("Removed " + remote + " from " + repo)
}
