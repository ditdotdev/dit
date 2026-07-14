// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const ghTestUserID = "11111111-1111-1111-1111-111111111111"

// ===========================================================================
// dit repo collaborator add
// ===========================================================================

func TestRepoCollaboratorAddCmd_Success(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/repos/myorg/myrepo/collaborators" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !contains(string(body), ghTestUserID) || !contains(string(body), `"permission":"write"`) {
			t.Errorf("unexpected body: %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"userId":"` + ghTestUserID + `","permission":"write"}`))
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execRepoCmd("repo", "collaborator", "add", "myorg", "myrepo", ghTestUserID,
		"--permission", "write", "--server", server.URL)
	if err != nil {
		t.Fatalf("collaborator add should succeed, got: %v", err)
	}
	if !contains(output, "write") {
		t.Errorf("output should mention permission, got: %s", output)
	}
}

func TestRepoCollaboratorAddCmd_Conflict(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "add", "myorg", "myrepo", ghTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator add conflict should error")
	}
}

func TestRepoCollaboratorAddCmd_NotFound(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "add", "myorg", "myrepo", ghTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator add not found should error")
	}
}

func TestRepoCollaboratorAddCmd_Forbidden(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "add", "myorg", "myrepo", ghTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator add forbidden should error")
	}
}

func TestRepoCollaboratorAddCmd_Unauthorized(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "add", "myorg", "myrepo", ghTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator add unauthorized should error")
	}
}

func TestRepoCollaboratorAddCmd_ServerError(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "add", "myorg", "myrepo", ghTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator add server error should error")
	}
}

func TestRepoCollaboratorAddCmd_NoAuth(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")
	cleanup := setupRepoEmptyCreds(t)
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "add", "myorg", "myrepo", ghTestUserID, "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("collaborator add without auth should error")
	}
}

func TestRepoCollaboratorAddCmd_ConnectionError(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	cleanup := setupRepoTestCreds(t, "http://127.0.0.1:1", "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "add", "myorg", "myrepo", ghTestUserID, "--server", "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("collaborator add connection error should error")
	}
}

// ===========================================================================
// dit repo collaborator remove
// ===========================================================================

func TestRepoCollaboratorRemoveCmd_Success(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/repos/myorg/myrepo/collaborators/"+ghTestUserID {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"removed"}`))
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execRepoCmd("repo", "collaborator", "rm", "myorg", "myrepo", ghTestUserID, "--server", server.URL)
	if err != nil {
		t.Fatalf("collaborator remove should succeed, got: %v", err)
	}
	if !contains(output, "Removed") {
		t.Errorf("output should confirm removal, got: %s", output)
	}
}

func TestRepoCollaboratorRemoveCmd_NotFound(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "remove", "myorg", "myrepo", ghTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator remove not found should error")
	}
}

func TestRepoCollaboratorRemoveCmd_Forbidden(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "remove", "myorg", "myrepo", ghTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator remove forbidden should error")
	}
}

func TestRepoCollaboratorRemoveCmd_Unauthorized(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "remove", "myorg", "myrepo", ghTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator remove unauthorized should error")
	}
}

func TestRepoCollaboratorRemoveCmd_ServerError(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "remove", "myorg", "myrepo", ghTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator remove server error should error")
	}
}

func TestRepoCollaboratorRemoveCmd_NoAuth(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")
	cleanup := setupRepoEmptyCreds(t)
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "remove", "myorg", "myrepo", ghTestUserID, "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("collaborator remove without auth should error")
	}
}

func TestRepoCollaboratorRemoveCmd_ConnectionError(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	cleanup := setupRepoTestCreds(t, "http://127.0.0.1:1", "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "remove", "myorg", "myrepo", ghTestUserID, "--server", "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("collaborator remove connection error should error")
	}
}

// ===========================================================================
// dit repo collaborator list
// ===========================================================================

func TestRepoCollaboratorListCmd_Success(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"userId": ghTestUserID, "permission": "admin"},
		})
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execRepoCmd("repo", "collaborator", "list", "myorg", "myrepo", "--server", server.URL)
	if err != nil {
		t.Fatalf("collaborator list should succeed, got: %v", err)
	}
	if !contains(output, ghTestUserID) || !contains(output, "admin") {
		t.Errorf("output should list collaborator, got: %s", output)
	}
}

func TestRepoCollaboratorListCmd_Empty(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execRepoCmd("repo", "collaborator", "ls", "myorg", "myrepo", "--server", server.URL)
	if err != nil {
		t.Fatalf("collaborator list empty should succeed, got: %v", err)
	}
	if !contains(output, "No collaborators found") {
		t.Errorf("empty list should show message, got: %s", output)
	}
}

func TestRepoCollaboratorListCmd_BadJSON(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "list", "myorg", "myrepo", "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator list bad JSON should error")
	}
}

func TestRepoCollaboratorListCmd_NotFound(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "list", "myorg", "myrepo", "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator list not found should error")
	}
}

func TestRepoCollaboratorListCmd_Forbidden(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "list", "myorg", "myrepo", "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator list forbidden should error")
	}
}

func TestRepoCollaboratorListCmd_Unauthorized(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "list", "myorg", "myrepo", "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator list unauthorized should error")
	}
}

func TestRepoCollaboratorListCmd_ServerError(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()
	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "list", "myorg", "myrepo", "--server", server.URL)
	if err == nil {
		t.Fatal("collaborator list server error should error")
	}
}

func TestRepoCollaboratorListCmd_NoAuth(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")
	cleanup := setupRepoEmptyCreds(t)
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "list", "myorg", "myrepo", "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("collaborator list without auth should error")
	}
}

func TestRepoCollaboratorListCmd_ConnectionError(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	cleanup := setupRepoTestCreds(t, "http://127.0.0.1:1", "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "collaborator", "list", "myorg", "myrepo", "--server", "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("collaborator list connection error should error")
	}
}

func TestRepoCollaboratorCmd_Help(t *testing.T) {
	resetRepoFlags()
	output, _ := execRepoCmd("repo", "collaborator", "--help")
	if !contains(output, "add") || !contains(output, "remove") {
		t.Errorf("collaborator help should list subcommands, got: %s", output)
	}
}
