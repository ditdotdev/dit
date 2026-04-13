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
	if err := os.Unsetenv("DATADATDAT_API_KEY"); err != nil {
		t.Fatalf("failed to unset env: %v", err)
	}

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
	if err := os.Unsetenv("DATADATDAT_API_KEY"); err != nil {
		t.Fatalf("failed to unset env: %v", err)
	}

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

func TestOrgListCmd_DefaultServer(t *testing.T) {
	resetOrgFlags()
	t.Setenv("DATADATDAT_API_KEY", "")
	if err := os.Unsetenv("DATADATDAT_API_KEY"); err != nil {
		t.Fatalf("failed to unset env: %v", err)
	}

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"name": "default-org"},
		})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origCredentialsPath := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origCredentialsPath }()

	// Store credentials with default server — no --server flag
	creds := common.Credentials{
		Servers: map[string]common.ServerCredential{
			server.URL: {APIKey: "default-key"},
		},
		DefaultServer: server.URL,
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"org", "list"}) // no --server flag

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("org list with default server error = %v", err)
	}
}

func TestOrgListCmd_NoServerConfigured(t *testing.T) {
	resetOrgFlags()
	t.Setenv("DATADATDAT_API_KEY", "some-key")

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origCredentialsPath := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origCredentialsPath }()

	// Empty credentials — no default server
	creds := common.Credentials{
		Servers:       map[string]common.ServerCredential{},
		DefaultServer: "",
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"org", "list"}) // no --server, no default

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("org list without server should return error")
	}
}

func TestOrgListCmd_ServerError(t *testing.T) {
	resetOrgFlags()
	t.Setenv("DATADATDAT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

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
	rootCmd.SetArgs([]string{"org", "list", "--server", server.URL})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("org list with server error should return error")
	}
}

func TestOrgListCmd_ServerUnauthorized(t *testing.T) {
	resetOrgFlags()
	t.Setenv("DATADATDAT_API_KEY", "bad-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origCredentialsPath := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origCredentialsPath }()

	creds := common.Credentials{
		Servers: map[string]common.ServerCredential{
			server.URL: {APIKey: "bad-key"},
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
	if err == nil {
		t.Fatal("org list with 401 should return error")
	}
}

func TestOrgListCmd_EmptyList(t *testing.T) {
	resetOrgFlags()
	t.Setenv("DATADATDAT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

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
	rootCmd.SetArgs([]string{"org", "list", "--server", server.URL})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("org list with empty list error = %v", err)
	}
}

func TestOrgListCmd_BadJSON(t *testing.T) {
	resetOrgFlags()
	t.Setenv("DATADATDAT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

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
	rootCmd.SetArgs([]string{"org", "list", "--server", server.URL})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("org list with bad JSON should return error")
	}
}

func TestOrgListCmd_LsAlias(t *testing.T) {
	resetOrgFlags()
	// Verify "org ls" works as alias for "org list"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, "[]")
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

// ===========================================================================
// helpers for org create / info / members tests
// ===========================================================================

func resetOrgCreateFlags() {
	_ = orgCreateCmd.Flags().Set("server", "")
	_ = orgCreateCmd.Flags().Set("display-name", "")
}

func resetOrgInfoFlags() {
	_ = orgInfoCmd.Flags().Set("server", "")
}

func resetOrgMembersFlags() {
	_ = orgMembersCmd.Flags().Set("server", "")
}

func setupOrgTestCreds(t *testing.T, serverURL, apiKey string) func() {
	t.Helper()
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origOverride := credentialsPathOverride
	credentialsPathOverride = credsFile

	creds := common.Credentials{
		Servers: map[string]common.ServerCredential{
			serverURL: {APIKey: apiKey},
		},
		DefaultServer: serverURL,
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	return func() { credentialsPathOverride = origOverride }
}

func setupOrgEmptyCreds(t *testing.T) func() {
	t.Helper()
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origOverride := credentialsPathOverride
	credentialsPathOverride = credsFile

	creds := common.Credentials{Servers: map[string]common.ServerCredential{}, DefaultServer: ""}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	return func() { credentialsPathOverride = origOverride }
}

func execOrgCmd(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

// ===========================================================================
// d3 org create
// ===========================================================================

func TestOrgCreateCmd_Success(t *testing.T) {
	resetOrgCreateFlags()
	t.Setenv("DATADATDAT_API_KEY", "")
	_ = os.Unsetenv("DATADATDAT_API_KEY")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/orgs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("bad auth header: %s", r.Header.Get("Authorization"))
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["name"] != "new-org" {
			t.Errorf("expected name=new-org, got %s", body["name"])
		}
		if body["displayName"] != "New Organization" {
			t.Errorf("expected displayName=New Organization, got %s", body["displayName"])
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"name":        "new-org",
			"displayName": "New Organization",
		})
	}))
	defer server.Close()

	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execOrgCmd("org", "create", "new-org", "--display-name", "New Organization", "--server", server.URL)
	if err != nil {
		t.Fatalf("org create should succeed, got error: %v", err)
	}
	if !contains(output, "new-org") {
		t.Errorf("output should mention created org, got: %s", output)
	}
}

func TestOrgCreateCmd_AlreadyExists(t *testing.T) {
	resetOrgCreateFlags()
	t.Setenv("DATADATDAT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"organization already exists"}`))
	}))
	defer server.Close()

	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "create", "existing-org", "--server", server.URL)
	if err == nil {
		t.Fatal("org create with 409 should return error")
	}
}

func TestOrgCreateCmd_NoAuth(t *testing.T) {
	resetOrgCreateFlags()
	t.Setenv("DATADATDAT_API_KEY", "")
	_ = os.Unsetenv("DATADATDAT_API_KEY")

	cleanup := setupOrgEmptyCreds(t)
	defer cleanup()

	_, err := execOrgCmd("org", "create", "new-org", "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("org create without auth should return error")
	}
}

func TestOrgCreateCmd_MissingArgs(t *testing.T) {
	resetOrgCreateFlags()

	_, err := execOrgCmd("org", "create")
	if err == nil {
		t.Fatal("org create without name should return error")
	}
}

func TestOrgCreateCmd_NoServer(t *testing.T) {
	resetOrgCreateFlags()
	t.Setenv("DATADATDAT_API_KEY", "some-key")

	cleanup := setupOrgEmptyCreds(t)
	defer cleanup()

	_, err := execOrgCmd("org", "create", "new-org")
	if err == nil {
		t.Fatal("org create without server should return error")
	}
}

// ===========================================================================
// d3 org info
// ===========================================================================

func TestOrgInfoCmd_Success(t *testing.T) {
	resetOrgInfoFlags()
	t.Setenv("DATADATDAT_API_KEY", "")
	_ = os.Unsetenv("DATADATDAT_API_KEY")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/orgs/test-org" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":        "test-org",
			"displayName": "Test Organization",
		})
	}))
	defer server.Close()

	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execOrgCmd("org", "info", "test-org", "--server", server.URL)
	if err != nil {
		t.Fatalf("org info should succeed, got error: %v", err)
	}
	if !contains(output, "test-org") {
		t.Errorf("output should contain org name, got: %s", output)
	}
	if !contains(output, "Test Organization") {
		t.Errorf("output should contain display name, got: %s", output)
	}
}

func TestOrgInfoCmd_NotFound(t *testing.T) {
	resetOrgInfoFlags()
	t.Setenv("DATADATDAT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "info", "nonexistent", "--server", server.URL)
	if err == nil {
		t.Fatal("org info with 404 should return error")
	}
}

func TestOrgInfoCmd_NoAuth(t *testing.T) {
	resetOrgInfoFlags()
	t.Setenv("DATADATDAT_API_KEY", "")
	_ = os.Unsetenv("DATADATDAT_API_KEY")

	cleanup := setupOrgEmptyCreds(t)
	defer cleanup()

	_, err := execOrgCmd("org", "info", "some-org", "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("org info without auth should return error")
	}
}

func TestOrgInfoCmd_MissingArgs(t *testing.T) {
	resetOrgInfoFlags()

	_, err := execOrgCmd("org", "info")
	if err == nil {
		t.Fatal("org info without name should return error")
	}
}

// ===========================================================================
// d3 org members
// ===========================================================================

func TestOrgMembersCmd_Success(t *testing.T) {
	resetOrgMembersFlags()
	t.Setenv("DATADATDAT_API_KEY", "")
	_ = os.Unsetenv("DATADATDAT_API_KEY")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/orgs/test-org/members" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"username": "alice", "role": "owner"},
			{"username": "bob", "role": "member"},
		})
	}))
	defer server.Close()

	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execOrgCmd("org", "members", "test-org", "--server", server.URL)
	if err != nil {
		t.Fatalf("org members should succeed, got error: %v", err)
	}
	if !contains(output, "alice") {
		t.Errorf("output should list members, got: %s", output)
	}
	if !contains(output, "owner") {
		t.Errorf("output should show roles, got: %s", output)
	}
}

func TestOrgMembersCmd_NotFound(t *testing.T) {
	resetOrgMembersFlags()
	t.Setenv("DATADATDAT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "members", "nonexistent", "--server", server.URL)
	if err == nil {
		t.Fatal("org members with 404 should return error")
	}
}

func TestOrgMembersCmd_NoAuth(t *testing.T) {
	resetOrgMembersFlags()
	t.Setenv("DATADATDAT_API_KEY", "")
	_ = os.Unsetenv("DATADATDAT_API_KEY")

	cleanup := setupOrgEmptyCreds(t)
	defer cleanup()

	_, err := execOrgCmd("org", "members", "some-org", "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("org members without auth should return error")
	}
}

func TestOrgMembersCmd_MissingArgs(t *testing.T) {
	resetOrgMembersFlags()

	_, err := execOrgCmd("org", "members")
	if err == nil {
		t.Fatal("org members without name should return error")
	}
}

func TestOrgMembersCmd_EmptyList(t *testing.T) {
	resetOrgMembersFlags()
	t.Setenv("DATADATDAT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execOrgCmd("org", "members", "empty-org", "--server", server.URL)
	if err != nil {
		t.Fatalf("org members empty should succeed, got error: %v", err)
	}
	if !contains(output, "No members found") {
		t.Errorf("empty members list should show message, got: %s", output)
	}
}
