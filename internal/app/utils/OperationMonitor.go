package utils

import (
	"context"
	"fmt"
	"github.com/antihax/optional"
	titanclient "github.com/datadatdat/titan-client-go"
	"strconv"
	"time"
)

const (
	OperationStateComplete = "COMPLETE"
	OperationStateFailed   = "FAILED"
	OperationStateAbort    = "ABORT"
)

var cfg = titanclient.NewConfiguration()
var apiClient = titanclient.NewAPIClient(cfg)
var operationsApi = apiClient.OperationsApi
var ctx = context.Background()

type operationMonitor struct {
	repo      string
	operation titanclient.Operation
}

func OperationMonitor(r string, o titanclient.Operation) operationMonitor {
	return operationMonitor{
		repo:      r,
		operation: o,
	}
}

func (om operationMonitor) IsTerminal(state string) bool {
	r := state == OperationStateFailed || state == OperationStateAbort || state == OperationStateComplete
	return r
}

func (om operationMonitor) Monitor(port int) bool {
	cfg.BasePath = "http://localhost:" + strconv.Itoa(port)

	padLen := 0
	//aborted := false
	state := "START"
	var lastId int32 = 0

	for !om.IsTerminal(state) {
		p := &titanclient.GetOperationProgressOpts{LastId: optional.NewInt32(lastId)}
		entries, _, err := operationsApi.GetOperationProgress(ctx, om.operation.Id, p)
		if err == nil {
			if len(entries) > 0 {
				state = entries[len(entries)-1].Type
			}
			for _, e := range entries {
				if e.Type != "PROGRESS" {
					if e.Message != "" {
						fmt.Println(e.Message)
					}
					padLen = 0
				} else {
					m := e.Message
					if len(m) > padLen {
						padLen = len(m)
					}
					fmt.Printf("\r%s", m[0:(padLen-len(m)+1)])
				}
				if e.Id > lastId {
					lastId = e.Id
				}
			}
			time.Sleep(2 * time.Second)
		} else {
			/**
			 * We swallow interrupts and instead translate them to an abort call. The operation may have already
			 * completed, so we swallow any exception there. If the users sends multiple interrupts (e.g.
			 * mashing Ctrl-C), then we let them exit out in case there's something seriously broken on the
			 * server.
			 */
			// Handle error by breaking the loop and returning false
			fmt.Printf("Error monitoring operation: %v\n", err)
			break
		}
	}

	var opText string
	if om.operation.Type == "PULL" {
		opText = "Pull"
	} else {
		opText = "Push"
	}
	switch state {
	case OperationStateComplete:
		fmt.Println(opText + " completed successfully")
	case OperationStateFailed:
		fmt.Println(opText + " failed")
	case OperationStateAbort:
		fmt.Println(opText + " aborted")
	}
	return state == OperationStateComplete
}
