package common

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	datadatdatclient "github.com/datadatdat/datadatdat-client-go"
)

// newGenericOpenAPIErrorForTest produces a real datadatdatclient.GenericOpenAPIError
// by spinning up an httptest server that mimics the datadatdat API returning the
// given HTTP status and JSON body, then invoking operationsApi.Pull against it.
// GenericOpenAPIError's fields are unexported, so this is the only reliable
// way to construct one in a test outside the client-go package.
func newGenericOpenAPIErrorForTest(t *testing.T, status int, apiErr datadatdatclient.ApiError) error {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		body := fmt.Sprintf(`{"code":%q,"message":%q,"details":%q}`, apiErr.GetCode(), apiErr.Message, apiErr.GetDetails())
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	c := datadatdatclient.NewConfiguration()
	c.Servers[0].URL = srv.URL
	client := datadatdatclient.NewAPIClient(c)
	params := datadatdatclient.RemoteParameters{Provider: "s3", Properties: map[string]any{}}
	_, _, err := client.OperationsApi.Pull(context.Background(), "repo", "origin", "commit").RemoteParameters(params).Execute()
	if err == nil {
		t.Fatalf("expected non-nil error from mock server")
	}
	return err
}

func TestHandleOperationError_Nil(t *testing.T) {
	handled := false
	output := captureStdout(func() {
		handled = handleOperationError(nil)
	})
	if handled {
		t.Errorf("handleOperationError(nil) should return false")
	}
	if output != "" {
		t.Errorf("handleOperationError(nil) should not print, got %q", output)
	}
}

func TestHandleOperationError_ApiErrorMessage(t *testing.T) {
	apiErr := datadatdatclient.ApiError{
		Message: "commit 'abc123' already exists in repository 'myrepo'",
	}
	apiErr.SetCode("CommitExists")
	err := newGenericOpenAPIErrorForTest(t, http.StatusConflict, apiErr)

	handled := false
	output := captureStdout(func() {
		handled = handleOperationError(err)
	})
	if !handled {
		t.Errorf("handleOperationError should return true on non-nil error")
	}
	if !strings.Contains(output, "commit 'abc123' already exists") {
		t.Errorf("expected server message in output, got %q", output)
	}
	if strings.Contains(output, "400 Bad Request") {
		t.Errorf("output must not leak misleading follow-on 400 error, got %q", output)
	}
	if strings.Contains(output, "monitoring operation") {
		t.Errorf("output must not say 'monitoring operation', got %q", output)
	}
}

func TestHandleOperationError_GenericError(t *testing.T) {
	err := fmt.Errorf("connection refused")
	handled := false
	output := captureStdout(func() {
		handled = handleOperationError(err)
	})
	if !handled {
		t.Errorf("handleOperationError should return true on non-nil error")
	}
	if !strings.Contains(output, "connection refused") {
		t.Errorf("expected original error in output, got %q", output)
	}
}

func TestHandleOperationError_ApiErrorEmptyMessage(t *testing.T) {
	apiErr := datadatdatclient.ApiError{}
	apiErr.SetCode("Unknown")
	err := newGenericOpenAPIErrorForTest(t, http.StatusInternalServerError, apiErr)

	handled := false
	output := captureStdout(func() {
		handled = handleOperationError(err)
	})
	if !handled {
		t.Errorf("handleOperationError should return true on non-nil error")
	}
	if strings.TrimSpace(output) == "" {
		t.Errorf("handleOperationError with empty ApiError.Message should still print something, got %q", output)
	}
}
