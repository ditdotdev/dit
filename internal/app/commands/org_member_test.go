// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package commands

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const orgTestUserID = "22222222-2222-2222-2222-222222222222"

func resetOrgMemberFlags() {
	_ = orgMemberAddCmd.Flags().Set("server", "")
	_ = orgMemberAddCmd.Flags().Set("role", "member")
	_ = orgMemberAddCmd.Flags().Set("github-login", "false")
	_ = orgMemberSetRoleCmd.Flags().Set("server", "")
	_ = orgMemberSetRoleCmd.Flags().Set("role", "")
	_ = orgMemberRemoveCmd.Flags().Set("server", "")
}

// ===========================================================================
// dit org member add
// ===========================================================================

func TestOrgMemberAddCmd_ByUserID(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/orgs/myorg/members" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !contains(string(body), `"userId":"`+orgTestUserID+`"`) || !contains(string(body), `"role":"admin"`) {
			t.Errorf("unexpected body: %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execOrgCmd("org", "member", "add", "myorg", orgTestUserID, "--role", "admin", "--server", server.URL)
	if err != nil {
		t.Fatalf("org member add should succeed, got: %v", err)
	}
	if !contains(output, "admin") {
		t.Errorf("output should mention role, got: %s", output)
	}
}

func TestOrgMemberAddCmd_ByGithubLogin(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !contains(string(body), `"githubLogin":"d3-ghtest2"`) {
			t.Errorf("expected githubLogin in body, got: %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "add", "myorg", "d3-ghtest2", "--github-login", "--server", server.URL)
	if err != nil {
		t.Fatalf("org member add --github-login should succeed, got: %v", err)
	}
}

func TestOrgMemberAddCmd_Conflict(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "add", "myorg", orgTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("org member add conflict should error")
	}
}

func TestOrgMemberAddCmd_NotFound(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "add", "myorg", orgTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("org member add not found should error")
	}
}

func TestOrgMemberAddCmd_Forbidden(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "add", "myorg", orgTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("org member add forbidden should error")
	}
}

func TestOrgMemberAddCmd_Unauthorized(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "add", "myorg", orgTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("org member add unauthorized should error")
	}
}

func TestOrgMemberAddCmd_ServerError(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "add", "myorg", orgTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("org member add server error should error")
	}
}

func TestOrgMemberAddCmd_NoAuth(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")
	cleanup := setupOrgEmptyCreds(t)
	defer cleanup()

	_, err := execOrgCmd("org", "member", "add", "myorg", orgTestUserID, "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("org member add without auth should error")
	}
}

func TestOrgMemberAddCmd_ConnectionError(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	cleanup := setupOrgTestCreds(t, "http://127.0.0.1:1", "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "add", "myorg", orgTestUserID, "--server", "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("org member add connection error should error")
	}
}

// ===========================================================================
// dit org member set-role
// ===========================================================================

func TestOrgMemberSetRoleCmd_Success(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/orgs/myorg/members/"+orgTestUserID {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !contains(string(body), `"role":"admin"`) {
			t.Errorf("unexpected body: %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execOrgCmd("org", "member", "set-role", "myorg", orgTestUserID, "--role", "admin", "--server", server.URL)
	if err != nil {
		t.Fatalf("org member set-role should succeed, got: %v", err)
	}
	if !contains(output, "admin") {
		t.Errorf("output should confirm role, got: %s", output)
	}
}

func TestOrgMemberSetRoleCmd_RequiresRole(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	cleanup := setupOrgTestCreds(t, "http://localhost:9999", "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "set-role", "myorg", orgTestUserID, "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("org member set-role without --role should error")
	}
}

func TestOrgMemberSetRoleCmd_NotFound(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "set-role", "myorg", orgTestUserID, "--role", "admin", "--server", server.URL)
	if err == nil {
		t.Fatal("org member set-role not found should error")
	}
}

func TestOrgMemberSetRoleCmd_Forbidden(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "set-role", "myorg", orgTestUserID, "--role", "admin", "--server", server.URL)
	if err == nil {
		t.Fatal("org member set-role forbidden should error")
	}
}

func TestOrgMemberSetRoleCmd_Unauthorized(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "set-role", "myorg", orgTestUserID, "--role", "admin", "--server", server.URL)
	if err == nil {
		t.Fatal("org member set-role unauthorized should error")
	}
}

func TestOrgMemberSetRoleCmd_ServerError(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "set-role", "myorg", orgTestUserID, "--role", "admin", "--server", server.URL)
	if err == nil {
		t.Fatal("org member set-role server error should error")
	}
}

func TestOrgMemberSetRoleCmd_NoAuth(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")
	cleanup := setupOrgEmptyCreds(t)
	defer cleanup()

	_, err := execOrgCmd("org", "member", "set-role", "myorg", orgTestUserID, "--role", "admin", "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("org member set-role without auth should error")
	}
}

func TestOrgMemberSetRoleCmd_ConnectionError(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	cleanup := setupOrgTestCreds(t, "http://127.0.0.1:1", "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "set-role", "myorg", orgTestUserID, "--role", "admin", "--server", "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("org member set-role connection error should error")
	}
}

// ===========================================================================
// dit org member remove
// ===========================================================================

func TestOrgMemberRemoveCmd_Success(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/orgs/myorg/members/"+orgTestUserID {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execOrgCmd("org", "member", "rm", "myorg", orgTestUserID, "--server", server.URL)
	if err != nil {
		t.Fatalf("org member remove should succeed, got: %v", err)
	}
	if !contains(output, "Removed") {
		t.Errorf("output should confirm removal, got: %s", output)
	}
}

func TestOrgMemberRemoveCmd_BadRequest(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("cannot remove last owner"))
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "remove", "myorg", orgTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("org member remove 400 should error")
	}
}

func TestOrgMemberRemoveCmd_NotFound(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "remove", "myorg", orgTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("org member remove not found should error")
	}
}

func TestOrgMemberRemoveCmd_Forbidden(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "remove", "myorg", orgTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("org member remove forbidden should error")
	}
}

func TestOrgMemberRemoveCmd_Unauthorized(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "remove", "myorg", orgTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("org member remove unauthorized should error")
	}
}

func TestOrgMemberRemoveCmd_ServerError(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()
	cleanup := setupOrgTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "remove", "myorg", orgTestUserID, "--server", server.URL)
	if err == nil {
		t.Fatal("org member remove server error should error")
	}
}

func TestOrgMemberRemoveCmd_NoAuth(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")
	cleanup := setupOrgEmptyCreds(t)
	defer cleanup()

	_, err := execOrgCmd("org", "member", "remove", "myorg", orgTestUserID, "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("org member remove without auth should error")
	}
}

func TestOrgMemberRemoveCmd_ConnectionError(t *testing.T) {
	resetOrgMemberFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	cleanup := setupOrgTestCreds(t, "http://127.0.0.1:1", "test-key")
	defer cleanup()

	_, err := execOrgCmd("org", "member", "remove", "myorg", orgTestUserID, "--server", "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("org member remove connection error should error")
	}
}

func TestOrgMemberCmd_Help(t *testing.T) {
	resetOrgMemberFlags()
	output, _ := execOrgCmd("org", "member", "--help")
	if !contains(output, "add") || !contains(output, "remove") || !contains(output, "set-role") {
		t.Errorf("org member help should list subcommands, got: %s", output)
	}
}
