// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFork_InvalidURL(t *testing.T) {
	// Should return an error on invalid URL, not panic
	if err := Fork("://invalid", "", ""); err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestFork_MissingOrgRepo(t *testing.T) {
	// URL without org/repo path should return an error, not panic
	if err := Fork("http://localhost:8080", "", ""); err == nil {
		t.Error("expected error for missing org/repo, got nil")
	}
}

func TestFork_NoCredentials(t *testing.T) {
	t.Setenv("DIT_API_KEY", "")
	// Create a temp credentials file that is empty
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	empty := Credentials{Servers: make(map[string]ServerCredential)}
	data, _ := json.Marshal(empty)
	_ = os.WriteFile(credsFile, data, 0600)

	origOverride := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origOverride }()

	// Should return an auth error, not panic
	if err := Fork("http://localhost:9999/myorg/myrepo", "", ""); err == nil {
		t.Error("expected auth error with no credentials, got nil")
	}
}

func TestFork_SuccessfulFork(t *testing.T) {
	t.Setenv("DIT_API_KEY", "")
	// Set up a mock HTTP server that responds to the fork request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expectedPath := "/api/v1/repos/sourceorg/sourcerepo/fork"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization header 'Bearer test-api-key', got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Decode the request body
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["targetNamespace"] != "myorg" {
			t.Errorf("expected targetNamespace 'myorg', got %s", body["targetNamespace"])
		}

		w.WriteHeader(http.StatusCreated)
		resp := map[string]interface{}{
			"org":        "myorg",
			"repo":       "sourcerepo",
			"forkedFrom": "sourceorg/sourcerepo",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Store credentials for the mock server
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	creds := Credentials{
		Servers: map[string]ServerCredential{
			server.URL: {APIKey: "test-api-key"},
		},
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	origOverride := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origOverride }()

	// Call Fork with the mock server URL
	if err := Fork(server.URL+"/sourceorg/sourcerepo", "myorg", ""); err != nil {
		t.Errorf("expected successful fork, got error: %v", err)
	}
}

func TestFork_WithCustomName(t *testing.T) {
	t.Setenv("DIT_API_KEY", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["targetName"] != "custom-name" {
			t.Errorf("expected targetName 'custom-name', got %s", body["targetName"])
		}
		if body["targetNamespace"] != "myorg" {
			t.Errorf("expected targetNamespace 'myorg', got %s", body["targetNamespace"])
		}

		w.WriteHeader(http.StatusCreated)
		resp := map[string]interface{}{
			"org":  "myorg",
			"repo": "custom-name",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	creds := Credentials{
		Servers: map[string]ServerCredential{
			server.URL: {APIKey: "test-key"},
		},
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	origOverride := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origOverride }()

	if err := Fork(server.URL+"/sourceorg/sourcerepo", "myorg", "custom-name"); err != nil {
		t.Errorf("expected successful fork with custom name, got error: %v", err)
	}
}

func TestFork_HostGatewayOverride(t *testing.T) {
	t.Setenv("DIT_API_KEY", "")
	// Mock server stands in for the host-visible gateway.
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		expectedPath := "/api/v1/repos/sourceorg/sourcerepo/fork"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"forkedFrom": "sourceorg/sourcerepo"})
	}))
	defer server.Close()

	// The remote URL uses a Docker-internal host that is NOT reachable and NOT the
	// mock server. DIT_HOST_GATEWAY must retarget the request to the mock server,
	// while credentials remain keyed on the original (URL) server.
	const originalServer = "http://dit-api-gateway:8080"
	t.Setenv("DIT_HOST_GATEWAY", server.URL)

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	creds := Credentials{
		Servers: map[string]ServerCredential{
			originalServer: {APIKey: "test-key"},
		},
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	origOverride := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origOverride }()

	if err := Fork(originalServer+"/sourceorg/sourcerepo", "myorg", ""); err != nil {
		t.Errorf("expected successful fork via override gateway, got error: %v", err)
	}

	// If the override worked, the mock gateway received the request; if creds stayed
	// keyed on the original server, the auth header carries the stored key.
	if gotAuth != "Bearer test-key" {
		t.Errorf("expected fork request to reach DIT_HOST_GATEWAY with creds from the original server; got Authorization %q", gotAuth)
	}
}

func TestFork_Conflict(t *testing.T) {
	t.Setenv("DIT_API_KEY", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	creds := Credentials{
		Servers: map[string]ServerCredential{
			server.URL: {APIKey: "test-key"},
		},
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	origOverride := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origOverride }()

	// Should return a conflict error, not panic
	if err := Fork(server.URL+"/sourceorg/sourcerepo", "", ""); err == nil {
		t.Error("expected conflict error, got nil")
	}
}

func TestFork_NotFound(t *testing.T) {
	t.Setenv("DIT_API_KEY", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	creds := Credentials{
		Servers: map[string]ServerCredential{
			server.URL: {APIKey: "test-key"},
		},
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	origOverride := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origOverride }()

	// Should return a not-found error, not panic
	if err := Fork(server.URL+"/sourceorg/sourcerepo", "", ""); err == nil {
		t.Error("expected not-found error, got nil")
	}
}
