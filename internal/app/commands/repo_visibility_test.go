package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// ===========================================================================
// dit repo create --private
// ===========================================================================

func TestRepoCreateCmd_Private(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")

	var patchedPrivate *bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/myorg/myrepo":
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/repos/myorg/myrepo/visibility":
			var body struct {
				IsPrivate bool `json:"isPrivate"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			patchedPrivate = &body.IsPrivate
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"updated","isPrivate":true}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execRepoCmd("repo", "create", "myorg", "myrepo", "--private", "--server", server.URL)
	if err != nil {
		t.Fatalf("repo create --private should succeed, got error: %v", err)
	}
	if patchedPrivate == nil || !*patchedPrivate {
		t.Errorf("expected visibility PATCH with isPrivate=true, got %v", patchedPrivate)
	}
	if !contains(output, "private") {
		t.Errorf("output should mention private, got: %s", output)
	}
}

func TestRepoCreateCmd_PrivateVisibilityFails(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			return
		}
		// visibility PATCH fails
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "create", "myorg", "myrepo", "--private", "--server", server.URL)
	if err == nil {
		t.Fatal("repo create --private with failing visibility should return error")
	}
}

// ===========================================================================
// dit repo set-visibility
// ===========================================================================

func TestRepoSetVisibilityCmd_Private(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/repos/myorg/myrepo/visibility" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != testBearerToken {
			t.Errorf("bad auth header: %s", r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		if !contains(string(body), `"isPrivate":true`) {
			t.Errorf("expected isPrivate=true body, got: %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"updated","isPrivate":true}`))
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execRepoCmd("repo", "set-visibility", "myorg", "myrepo", "--private", "--server", server.URL)
	if err != nil {
		t.Fatalf("set-visibility --private should succeed, got error: %v", err)
	}
	if !contains(output, "private") {
		t.Errorf("output should mention private, got: %s", output)
	}
}

func TestRepoSetVisibilityCmd_Public(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !contains(string(body), `"isPrivate":false`) {
			t.Errorf("expected isPrivate=false body, got: %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"updated","isPrivate":false}`))
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	output, err := execRepoCmd("repo", "set-visibility", "myorg", "myrepo", "--public", "--server", server.URL)
	if err != nil {
		t.Fatalf("set-visibility --public should succeed, got error: %v", err)
	}
	if !contains(output, "public") {
		t.Errorf("output should mention public, got: %s", output)
	}
}

func TestRepoSetVisibilityCmd_RequiresExactlyOne(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")
	cleanup := setupRepoTestCreds(t, "http://localhost:9999", "test-key")
	defer cleanup()

	// Neither flag.
	_, err := execRepoCmd("repo", "set-visibility", "myorg", "myrepo", "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("set-visibility with neither flag should error")
	}

	resetRepoFlags()
	// Both flags.
	_, err = execRepoCmd("repo", "set-visibility", "myorg", "myrepo", "--private", "--public", "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("set-visibility with both flags should error")
	}
}

func TestRepoSetVisibilityCmd_NoAuth(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "")
	_ = os.Unsetenv("DIT_API_KEY")

	cleanup := setupRepoEmptyCreds(t)
	defer cleanup()

	_, err := execRepoCmd("repo", "set-visibility", "myorg", "myrepo", "--private", "--server", "http://localhost:9999")
	if err == nil {
		t.Fatal("set-visibility without auth should return error")
	}
}

func TestRepoSetVisibilityCmd_NotFound(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "set-visibility", "myorg", "myrepo", "--private", "--server", server.URL)
	if err == nil {
		t.Fatal("set-visibility on missing repo should return error")
	}
}

func TestRepoSetVisibilityCmd_Forbidden(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "set-visibility", "myorg", "myrepo", "--public", "--server", server.URL)
	if err == nil {
		t.Fatal("set-visibility forbidden should return error")
	}
}

func TestRepoSetVisibilityCmd_Unauthorized(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "set-visibility", "myorg", "myrepo", "--private", "--server", server.URL)
	if err == nil {
		t.Fatal("set-visibility unauthorized should return error")
	}
}

func TestRepoSetVisibilityCmd_ServerError(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream"))
	}))
	defer server.Close()

	cleanup := setupRepoTestCreds(t, server.URL, "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "set-visibility", "myorg", "myrepo", "--private", "--server", server.URL)
	if err == nil {
		t.Fatal("set-visibility server error should return error")
	}
}

func TestRepoSetVisibilityCmd_ConnectionError(t *testing.T) {
	resetRepoFlags()
	t.Setenv("DIT_API_KEY", "test-key")

	// Server URL points nowhere — connection refused exercises the transport error path.
	cleanup := setupRepoTestCreds(t, "http://127.0.0.1:1", "test-key")
	defer cleanup()

	_, err := execRepoCmd("repo", "set-visibility", "myorg", "myrepo", "--private", "--server", "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("set-visibility connection error should return error")
	}
}

func TestRepoSetVisibilityCmd_MissingArgs(t *testing.T) {
	resetRepoFlags()
	_, err := execRepoCmd("repo", "set-visibility", "only-org")
	if err == nil {
		t.Fatal("set-visibility with one arg should error")
	}
}
