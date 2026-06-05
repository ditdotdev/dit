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

	"github.com/ditdotdev/dit/internal/app/providers/common"
)

const testBearerToken = "Bearer test-key"

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func resetRepoFlags() {
	_ = repoCreateCmd.Flags().Set("server", "")
	_ = repoCreateCmd.Flags().Set("private", "false")
	_ = repoDeleteCmd.Flags().Set("server", "")
	_ = repoDeleteCmd.Flags().Set("force", "false")
	_ = repoListCmd.Flags().Set("server", "")
	_ = repoListCmd.Flags().Set("org", "")
	_ = repoSetVisibilityCmd.Flags().Set("server", "")
	_ = repoSetVisibilityCmd.Flags().Set("private", "false")
	_ = repoSetVisibilityCmd.Flags().Set("public", "false")
	_ = repoCollaboratorAddCmd.Flags().Set("server", "")
	_ = repoCollaboratorAddCmd.Flags().Set("permission", "read")
	_ = repoCollaboratorRemoveCmd.Flags().Set("server", "")
	_ = repoCollaboratorListCmd.Flags().Set("server", "")
}

// setupRepoTestCreds writes a temporary credentials file and sets the override.
// It returns a cleanup function that restores the original override.
func setupRepoTestCreds(t *testing.T, serverURL, apiKey string) func() {
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

// setupRepoEmptyCreds writes an empty credentials file (no servers).
func setupRepoEmptyCreds(t *testing.T) func() {
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

func execRepoCmd(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

// ===========================================================================
// dit repo create
// ===========================================================================

func TestRepoCreateCmd_Success(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/repos/myorg/myrepo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != testBearerToken {
			t.Errorf("bad auth header: %s", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"org":"myorg","repo":"myrepo"}`))
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execRepoCmd("repo", "create", "myorg", "myrepo", "--server", server.URL)
	if err != nil {
		t.Fatalf("repo create should succeed, got error: %v", err)
	}
	if !contains(output, "myorg/myrepo") {
		t.Errorf("output should mention created repo, got: %s", output)
	}
}

func TestRepoCreateCmd_AlreadyExists(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"repository already exists"}`))
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "create", "myorg", "myrepo", "--server", server.URL)
	if err == nil {
		t.Fatal("repo create with 409 should return error")
	}
}

func TestRepoCreateCmd_NoAuth(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")

	cleanup := setupRepoEmptyCreds(t)
	defer cleanup()

	_, err := execRepoCmd("repo", "create", "myorg", "myrepo", "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("repo create without auth should return error")
	}
}

func TestRepoCreateCmd_ServerError(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "create", "myorg", "myrepo", "--server", server.URL)
	if err == nil {
		t.Fatal("repo create with server error should return error")
	}
}

func TestRepoCreateCmd_NoServer(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "some-key")

	cleanup := setupRepoEmptyCreds(t)
	defer cleanup()

	_, err := execRepoCmd("repo", "create", "myorg", "myrepo")
	if err == nil {
		t.Fatal("repo create without server should return error")
	}
}

func TestRepoCreateCmd_MissingArgs(t *testing.T) {
	resetRepoFlags()

	_, err := execRepoCmd("repo", "create")
	if err == nil {
		t.Fatal("repo create without args should return error")
	}

	_, err = execRepoCmd("repo", "create", "only-org")
	if err == nil {
		t.Fatal("repo create with only one arg should return error")
	}
}

// ===========================================================================
// dit repo delete
// ===========================================================================

func TestRepoDeleteCmd_Success(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/repos/myorg/myrepo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != testBearerToken {
			t.Errorf("bad auth header: %s", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execRepoCmd("repo", "delete", "myorg", "myrepo", "--server", server.URL)
	if err != nil {
		t.Fatalf("repo delete should succeed, got error: %v", err)
	}
	if !contains(output, "myorg/myrepo") {
		t.Errorf("output should mention deleted repo, got: %s", output)
	}
}

func TestRepoDeleteCmd_NotFound(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "delete", "myorg", "myrepo", "--server", server.URL)
	if err == nil {
		t.Fatal("repo delete with 404 should return error")
	}
}

func TestRepoDeleteCmd_NoAuth(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")

	cleanup := setupRepoEmptyCreds(t)
	defer cleanup()

	_, err := execRepoCmd("repo", "delete", "myorg", "myrepo", "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("repo delete without auth should return error")
	}
}

func TestRepoDeleteCmd_MissingArgs(t *testing.T) {
	resetRepoFlags()

	_, err := execRepoCmd("repo", "delete")
	if err == nil {
		t.Fatal("repo delete without args should return error")
	}

	_, err = execRepoCmd("repo", "delete", "only-org")
	if err == nil {
		t.Fatal("repo delete with only one arg should return error")
	}
}

// ===========================================================================
// dit repo ls
// ===========================================================================

func TestRepoListCmd_Success(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/repos" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != testBearerToken {
			t.Errorf("bad auth header: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"org": "org1", "repo": "repo-a"},
			{"org": "org1", "repo": "repo-b"},
			{"org": "org2", "repo": "repo-c"},
		})
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execRepoCmd("repo", "ls", "--server", server.URL)
	if err != nil {
		t.Fatalf("repo ls should succeed, got error: %v", err)
	}
	if !contains(output, "org1/repo-a") {
		t.Errorf("output should list repos, got: %s", output)
	}
	if !contains(output, "org2/repo-c") {
		t.Errorf("output should list all repos, got: %s", output)
	}
}

func TestRepoListCmd_FilterByOrg(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/myorg" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"org": "myorg", "repo": "repo-a"},
		})
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execRepoCmd("repo", "ls", "--org", "myorg", "--server", server.URL)
	if err != nil {
		t.Fatalf("repo ls --org should succeed, got error: %v", err)
	}
	if !contains(output, "myorg/repo-a") {
		t.Errorf("output should list org repos, got: %s", output)
	}
}

func TestRepoListCmd_EmptyList(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execRepoCmd("repo", "ls", "--server", server.URL)
	if err != nil {
		t.Fatalf("repo ls empty should succeed, got error: %v", err)
	}
	if !contains(output, "No repositories found") {
		t.Errorf("empty list should show message, got: %s", output)
	}
}

func TestRepoListCmd_NoAuth(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")

	cleanup := setupRepoEmptyCreds(t)
	defer cleanup()

	_, err := execRepoCmd("repo", "ls", "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("repo ls without auth should return error")
	}
}

func TestRepoListCmd_ServerError(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "ls", "--server", server.URL)
	if err == nil {
		t.Fatal("repo ls with server error should return error")
	}
}

func TestRepoListCmd_Usage(t *testing.T) {
	resetRepoFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"repo", "--help"})
	_ = rootCmd.Execute()
	output := buf.String()
	if !contains(output, "create") {
		t.Errorf("repo help should list create subcommand, got: %s", output)
	}
	if !contains(output, "delete") {
		t.Errorf("repo help should list delete subcommand, got: %s", output)
	}
	// "list" or "ls" should appear
	if !contains(output, "list") && !contains(output, "ls") {
		t.Errorf("repo help should list ls subcommand, got: %s", output)
	}
}

func TestRepoListCmd_BadJSON(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, "not json")
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "ls", "--server", server.URL)
	if err == nil {
		t.Fatal("repo ls with bad JSON should return error")
	}
}
