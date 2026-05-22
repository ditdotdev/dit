package common

import (
	"fmt"
	"strconv"
)

func Abort(repo string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	var operations, _, _ = operationsApi.ListOperations(ctx).Execute() //TODO handle error
	var abortCount = 0
	for _, operation := range operations {
		if operation.State == "RUNNING" {
			fmt.Println("aborting operation " + operation.Id)
			if _, err := operationsApi.AbortOperation(ctx, operation.Id).Execute(); err != nil {
				fmt.Printf("Warning: Failed to abort operation %s: %v\n", operation.Id, err)
			}
			abortCount++
		}
	}
	if abortCount == 0 {
		fmt.Println("no operation in progress")
		osExit(0)
	}
}
