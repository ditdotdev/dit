// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"fmt"
	"strconv"
)

func Abort(repo string, port int) {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	operations, _, err := operationsApi.ListOperations(ctx).Execute()
	if err != nil {
		fmt.Println("Error listing operations: " + err.Error())
		osExit(1)
	}
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
