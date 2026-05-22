package utils

import (
	"context"
	"fmt"
	datadatdatclient "github.com/datadatdat/datadatdat-client-go"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	OperationStateComplete = "COMPLETE"
	OperationStateFailed   = "FAILED"
	OperationStateAbort    = "ABORT"
)

var cfg = datadatdatclient.NewConfiguration()
var apiClient = datadatdatclient.NewAPIClient(cfg)
var operationsApi = apiClient.OperationsApi
var ctx = context.Background()

type operationMonitor struct {
	repo      string
	operation datadatdatclient.Operation
}

func OperationMonitor(r string, o datadatdatclient.Operation) operationMonitor {
	return operationMonitor{
		repo:      r,
		operation: o,
	}
}

func (om operationMonitor) IsTerminal(state string) bool {
	r := state == OperationStateFailed || state == OperationStateAbort || state == OperationStateComplete
	return r
}

// formatProgressLine returns "\r" + msg left-padded with spaces to padLen
// so a shorter follow-up message overwrites the tail of the previous one.
// Pre-fix this used msg[0:(padLen-len(msg)+1)] which panicked with
// out-of-range when the next message was shorter than padLen.
func formatProgressLine(msg string, padLen int) string {
	return fmt.Sprintf("\r%-*s", padLen, msg)
}

func (om operationMonitor) Monitor(port int) bool {
	cfg.Servers[0].URL = "http://localhost:" + strconv.Itoa(port)

	// Translate SIGINT/SIGTERM into an abort RPC for the in-flight
	// operation. Pre-fix, the comment below claimed this behavior but
	// there was no signal handler — Ctrl-C killed the CLI and left the
	// server-side push/pull orphaned (post Phase-5 it's marked FAILED on
	// restart, but the user still wasted the in-flight work).
	//
	// First interrupt: best-effort abort + exit cleanly when the next
	// progress poll observes ABORT/FAILED/COMPLETE.
	// Second interrupt: hard exit (server may be hung; user can recover
	// from the FAILED state).
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	var interrupts atomic.Int32
	go func() {
		for sig := range sigCh {
			n := interrupts.Add(1)
			if n == 1 {
				fmt.Fprintf(os.Stderr, "\nReceived %v — requesting abort. Ctrl-C again to exit immediately.\n", sig)
				// Best-effort: ignore errors. The next poll loop iteration
				// will see ABORT in the progress stream when the server
				// acts on it.
				_, _ = operationsApi.AbortOperation(ctx, om.operation.Id).Execute()
			} else {
				fmt.Fprintln(os.Stderr, "Second interrupt — exiting without confirming abort.")
				os.Exit(130) // 128 + SIGINT(2), the conventional "killed by Ctrl-C" exit code
			}
		}
	}()

	padLen := 0
	state := "START"
	var lastId int32 = 0

	for !om.IsTerminal(state) {
		entries, _, err := operationsApi.GetOperationProgress(ctx, om.operation.Id).LastId(lastId).Execute()
		if err == nil {
			if len(entries) > 0 {
				state = entries[len(entries)-1].Type
			}
			for _, e := range entries {
				msg := e.GetMessage()
				if e.Type != "PROGRESS" {
					if msg != "" {
						fmt.Println(msg)
					}
					padLen = 0
				} else {
					padLen = max(padLen, len(msg))
					fmt.Print(formatProgressLine(msg, padLen))
				}
				if e.Id > lastId {
					lastId = e.Id
				}
			}
			time.Sleep(2 * time.Second)
		} else {
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
