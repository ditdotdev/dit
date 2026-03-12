package common

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestHandleRemoteError_UnauthorizedWithServer(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusUnauthorized}
	err := fmt.Errorf("401 Unauthorized")

	output := captureStdout(func() {
		handleRemoteError(err, resp, "http://api-gateway:8080")
	})

	if !strings.Contains(output, "--api-key") {
		t.Errorf("auth error message should include --api-key flag, got %q", output)
	}
	if !strings.Contains(output, "--server http://api-gateway:8080") {
		t.Errorf("auth error message should include --server flag, got %q", output)
	}
}

func TestHandleRemoteError_UnauthorizedWithoutServer(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusUnauthorized}
	err := fmt.Errorf("401 Unauthorized")

	output := captureStdout(func() {
		handleRemoteError(err, resp, "")
	})

	if !strings.Contains(output, "--api-key") {
		t.Errorf("auth error message should include --api-key flag, got %q", output)
	}
	if !strings.Contains(output, "d3 auth login") {
		t.Errorf("auth error message should include 'd3 auth login', got %q", output)
	}
}

func TestHandleRemoteError_NonAuthError(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusInternalServerError}
	err := fmt.Errorf("server error")

	output := captureStdout(func() {
		handleRemoteError(err, resp, "http://api-gateway:8080")
	})

	if strings.Contains(output, "auth") {
		t.Errorf("non-auth error should not mention auth, got %q", output)
	}
	if !strings.Contains(output, "server error") {
		t.Errorf("non-auth error should show original error, got %q", output)
	}
}

func TestHandleRemoteError_NilResponse(t *testing.T) {
	err := fmt.Errorf("connection refused")

	output := captureStdout(func() {
		handleRemoteError(err, nil, "http://api-gateway:8080")
	})

	if !strings.Contains(output, "connection refused") {
		t.Errorf("nil response should show original error, got %q", output)
	}
}
