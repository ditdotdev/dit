package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"datadatdat/internal/app/providers/common"
)

// resetOrgFlags clears flag state between tests since cobra retains values.
func resetOrgFlags() {
	_ = orgListCmd.Flags().Set("server", "")
}

func TestOrgListCmd_DisplaysOrgs(t *testing.T) {
	resetOrgFlags()
	// Ensure env var doesn't interfere — must use stored credentials
	t.Setenv("DATADATDAT_API_KEY", "")
	os.Unsetenv("DATADATDAT_API_KEY")

	// Mock server that returns a list of orgs
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/orgs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing or wrong Authorization header: %s", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"name": "my-org", "display_name": "My Org"},
			{"name": "other-org", "display_name": "Other Org"},
		})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origCredentialsPath := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origCredentialsPath }()

	// Store credentials pointing to mock server
	creds := common.Credentials{
		Servers: map[string]common.ServerCredential{
			server.URL: {APIKey: "test-key"},
		},
		DefaultServer: server.URL,
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"org", "list", "--server", server.URL})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("org list command error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("org list should produce output")
	}
}

func TestOrgListCmd_WithEnvVar(t *testing.T) {
	resetOrgFlags()
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer env-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"name": "env-org"},
		})
	}))
	defer server.Close()

	t.Setenv("DATADATDAT_API_KEY", "env-key")

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origCredentialsPath := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origCredentialsPath }()

	// Store credentials with server URL but different key (env should override)
	creds := common.Credentials{
		Servers: map[string]common.ServerCredential{
			server.URL: {APIKey: "stored-key"},
		},
		DefaultServer: server.URL,
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"org", "list", "--server", server.URL})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("org list with env var error = %v", err)
	}
}

func TestOrgListCmd_NoAuth(t *testing.T) {
	resetOrgFlags()
	t.Setenv("DATADATDAT_API_KEY", "")
	os.Unsetenv("DATADATDAT_API_KEY")

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origCredentialsPath := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origCredentialsPath }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"org", "list", "--server", "http://localhost:9999"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("org list without auth should return error")
	}
}

func TestOrgListCmd_LsAlias(t *testing.T) {
	resetOrgFlags()
	// Verify "org ls" works as alias for "org list"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "[]")
	}))
	defer server.Close()

	t.Setenv("DATADATDAT_API_KEY", "test-key")

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origCredentialsPath := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origCredentialsPath }()

	creds := common.Credentials{
		Servers: map[string]common.ServerCredential{
			server.URL: {APIKey: "test-key"},
		},
		DefaultServer: server.URL,
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"org", "ls", "--server", server.URL})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("org ls alias error = %v", err)
	}
}
