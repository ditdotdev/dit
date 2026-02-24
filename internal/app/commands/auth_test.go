package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"datadatdat/internal/app/providers/common"
)

// resetAuthFlags clears flag state between tests since cobra retains values.
func resetAuthFlags() {
	_ = authLoginCmd.Flags().Set("api-key", "")
	_ = authLoginCmd.Flags().Set("server", "")
	_ = authLogoutCmd.Flags().Set("server", "")
}

func TestAuthLoginCmd_StoresCredentials(t *testing.T) {
	resetAuthFlags()
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origCredentialsPath := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origCredentialsPath }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"auth", "login", "--api-key", "test-key-123", "--server", "http://localhost:8080"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth login command error = %v", err)
	}

	creds, err := common.LoadCredentials(credsFile)
	if err != nil {
		t.Fatalf("LoadCredentials() error = %v", err)
	}
	if creds.Servers["http://localhost:8080"].APIKey != "test-key-123" {
		t.Errorf("Stored APIKey = %q, want %q", creds.Servers["http://localhost:8080"].APIKey, "test-key-123")
	}
	if creds.DefaultServer != "http://localhost:8080" {
		t.Errorf("DefaultServer = %q, want %q", creds.DefaultServer, "http://localhost:8080")
	}
}

func TestAuthLoginCmd_RequiresAPIKey(t *testing.T) {
	resetAuthFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"auth", "login", "--server", "http://localhost:8080"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth login without --api-key should return error")
	}
}

func TestAuthStatusCmd_Authenticated(t *testing.T) {
	resetAuthFlags()
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origCredentialsPath := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origCredentialsPath }()

	// Store credentials first
	creds := common.Credentials{
		Servers: map[string]common.ServerCredential{
			"http://localhost:8080": {APIKey: "test-key-456"},
		},
		DefaultServer: "http://localhost:8080",
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"auth", "status"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth status command error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("auth status should produce output")
	}
}

func TestAuthStatusCmd_NotAuthenticated(t *testing.T) {
	resetAuthFlags()
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origCredentialsPath := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origCredentialsPath }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"auth", "status"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth status command error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("auth status should produce output even when not authenticated")
	}
}

func TestAuthLogoutCmd_RemovesCredentials(t *testing.T) {
	resetAuthFlags()
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origCredentialsPath := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origCredentialsPath }()

	// Store credentials first
	creds := common.Credentials{
		Servers: map[string]common.ServerCredential{
			"http://localhost:8080": {APIKey: "test-key-789"},
		},
		DefaultServer: "http://localhost:8080",
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"auth", "logout", "--server", "http://localhost:8080"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth logout command error = %v", err)
	}

	loaded, _ := common.LoadCredentials(credsFile)
	if _, exists := loaded.Servers["http://localhost:8080"]; exists {
		t.Error("auth logout did not remove credentials for the server")
	}
}

func TestAuthLogoutCmd_NoCredentialsIsNoOp(t *testing.T) {
	resetAuthFlags()
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	origCredentialsPath := credentialsPathOverride
	credentialsPathOverride = credsFile
	defer func() { credentialsPathOverride = origCredentialsPath }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"auth", "logout", "--server", "http://localhost:8080"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth logout without stored credentials should not error, got %v", err)
	}
}
