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
	// Should not panic on invalid URL
	Fork("://invalid", "", "")
}

func TestFork_MissingOrgRepo(t *testing.T) {
	// URL without org/repo path should print error, not panic
	Fork("http://localhost:8080", "", "")
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

	// Should print auth error, not panic
	Fork("http://localhost:9999/myorg/myrepo", "", "")
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
	Fork(server.URL+"/sourceorg/sourcerepo", "myorg", "")
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

	Fork(server.URL+"/sourceorg/sourcerepo", "myorg", "custom-name")
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

	// Should print conflict error, not panic
	Fork(server.URL+"/sourceorg/sourcerepo", "", "")
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

	// Should print not found error, not panic
	Fork(server.URL+"/sourceorg/sourcerepo", "", "")
}
