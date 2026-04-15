package common

import (
	"fmt"

	datadatdatclient "github.com/datadatdat/datadatdat-client-go"
)

// handleOperationError prints a user-facing error message when an operations
// API call fails. Returns true if err was non-nil (the caller should abort).
//
// For server-returned errors (GenericOpenAPIError wrapping an ApiError model),
// prints the server's message verbatim — e.g. "commit X already exists in
// repository Y". For empty messages, falls back to the wire-level status
// string. For any other error type, prints the error as-is.
//
// This exists because operationsApi.Pull/Push etc. were historically called
// with `_` for the error return, causing the CLI to proceed into
// OperationMonitor with an empty operation ID and surface a misleading
// "400 Bad Request" follow-on error. See issue #98.
func handleOperationError(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(datadatdatclient.GenericOpenAPIError); ok {
		if m, ok := e.Model().(datadatdatclient.ApiError); ok && m.Message != "" {
			fmt.Println(m.Message)
			return true
		}
		fmt.Printf("operation failed: %s\n", e.Error())
		return true
	}
	fmt.Printf("operation failed: %v\n", err)
	return true
}
