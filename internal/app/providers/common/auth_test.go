package common

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetAPIKey_EnvVarOverride(t *testing.T) {
	// DATADATDAT_API_KEY env var should take priority over stored credentials
	t.Setenv("DATADATDAT_API_KEY", "env-key-123")

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	creds := Credentials{
		Servers: map[string]ServerCredential{
			"http://localhost:8080": {APIKey: "stored-key-456"},
		},
		DefaultServer: "http://localhost:8080",
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	key := GetAPIKey(credsFile)
	if key != "env-key-123" {
		t.Errorf("GetAPIKey() = %q, want %q (env var should override stored)", key, "env-key-123")
	}
}

func TestGetAPIKey_FallsBackToStored(t *testing.T) {
	t.Setenv("DATADATDAT_API_KEY", "")
	if err := os.Unsetenv("DATADATDAT_API_KEY"); err != nil {
		t.Fatalf("failed to unset env: %v", err)
	}

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	creds := Credentials{
		Servers: map[string]ServerCredential{
			"http://localhost:8080": {APIKey: "stored-key-456"},
		},
		DefaultServer: "http://localhost:8080",
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	key := GetAPIKey(credsFile)
	if key != "stored-key-456" {
		t.Errorf("GetAPIKey() = %q, want %q", key, "stored-key-456")
	}
}

func TestGetAPIKey_NoCredentials(t *testing.T) {
	t.Setenv("DATADATDAT_API_KEY", "")
	if err := os.Unsetenv("DATADATDAT_API_KEY"); err != nil {
		t.Fatalf("failed to unset env: %v", err)
	}

	key := GetAPIKey("/nonexistent/credentials")
	if key != "" {
		t.Errorf("GetAPIKey() = %q, want empty string", key)
	}
}

func TestLoadCredentials_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	expected := Credentials{
		Servers: map[string]ServerCredential{
			"http://localhost:8080": {APIKey: "my-key"},
		},
		DefaultServer: "http://localhost:8080",
	}
	data, _ := json.Marshal(expected)
	_ = os.WriteFile(credsFile, data, 0600)

	creds, err := LoadCredentials(credsFile)
	if err != nil {
		t.Fatalf("LoadCredentials() error = %v", err)
	}
	if creds.DefaultServer != expected.DefaultServer {
		t.Errorf("DefaultServer = %q, want %q", creds.DefaultServer, expected.DefaultServer)
	}
	if creds.Servers["http://localhost:8080"].APIKey != "my-key" {
		t.Errorf("APIKey = %q, want %q", creds.Servers["http://localhost:8080"].APIKey, "my-key")
	}
}

func TestLoadCredentials_FileNotExist(t *testing.T) {
	creds, err := LoadCredentials("/nonexistent/credentials")
	if err != nil {
		t.Fatalf("LoadCredentials() error = %v, want nil for missing file", err)
	}
	if creds.Servers == nil {
		t.Fatal("LoadCredentials() returned nil Servers map")
	}
	if len(creds.Servers) != 0 {
		t.Errorf("LoadCredentials() returned %d servers, want 0", len(creds.Servers))
	}
}

func TestSaveCredentials_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")

	creds := Credentials{
		Servers: map[string]ServerCredential{
			"http://localhost:8080": {APIKey: "saved-key"},
		},
		DefaultServer: "http://localhost:8080",
	}

	err := SaveCredentials(credsFile, creds)
	if err != nil {
		t.Fatalf("SaveCredentials() error = %v", err)
	}

	// Verify file was written
	loaded, err := LoadCredentials(credsFile)
	if err != nil {
		t.Fatalf("LoadCredentials() after save error = %v", err)
	}
	if loaded.Servers["http://localhost:8080"].APIKey != "saved-key" {
		t.Errorf("Loaded APIKey = %q, want %q", loaded.Servers["http://localhost:8080"].APIKey, "saved-key")
	}

	// Verify file permissions (non-Windows only; Windows ignores Unix permission bits)
	if os.Getenv("OS") != "Windows_NT" {
		info, _ := os.Stat(credsFile)
		if info.Mode().Perm()&0077 != 0 {
			t.Errorf("Credentials file permissions = %o, want owner-only access", info.Mode().Perm())
		}
	}
}

func TestSaveCredentials_CreatesParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "subdir", "credentials")

	creds := Credentials{
		Servers:       map[string]ServerCredential{},
		DefaultServer: "",
	}

	err := SaveCredentials(credsFile, creds)
	if err != nil {
		t.Fatalf("SaveCredentials() error = %v", err)
	}

	if _, err := os.Stat(credsFile); os.IsNotExist(err) {
		t.Error("SaveCredentials() did not create the file")
	}
}

func TestCredentialsPath(t *testing.T) {
	path := CredentialsPath()
	if path == "" {
		t.Fatal("CredentialsPath() returned empty string")
	}
	// Should end with .datadatdat/credentials
	if filepath.Base(path) != "credentials" {
		t.Errorf("CredentialsPath() = %q, want to end with 'credentials'", path)
	}
	dir := filepath.Base(filepath.Dir(path))
	if dir != ".datadatdat" {
		t.Errorf("CredentialsPath() parent dir = %q, want '.datadatdat'", dir)
	}
}

func TestGetAPIKeyForServer(t *testing.T) {
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	creds := Credentials{
		Servers: map[string]ServerCredential{
			"http://server-a:8080": {APIKey: "key-a"},
			"http://server-b:8080": {APIKey: "key-b"},
		},
		DefaultServer: "http://server-a:8080",
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	tests := []struct {
		name   string
		server string
		want   string
	}{
		{"specific server", "http://server-a:8080", "key-a"},
		{"other server", "http://server-b:8080", "key-b"},
		{"unknown server", "http://server-c:8080", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GetAPIKeyForServer(credsFile, tt.server)
			if key != tt.want {
				t.Errorf("GetAPIKeyForServer(%q) = %q, want %q", tt.server, key, tt.want)
			}
		})
	}
}

func TestRemoveCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	creds := Credentials{
		Servers: map[string]ServerCredential{
			"http://server-a:8080": {APIKey: "key-a"},
			"http://server-b:8080": {APIKey: "key-b"},
		},
		DefaultServer: "http://server-a:8080",
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	err := RemoveCredentials(credsFile, "http://server-a:8080")
	if err != nil {
		t.Fatalf("RemoveCredentials() error = %v", err)
	}

	loaded, _ := LoadCredentials(credsFile)
	if _, exists := loaded.Servers["http://server-a:8080"]; exists {
		t.Error("RemoveCredentials() did not remove the server entry")
	}
	if loaded.Servers["http://server-b:8080"].APIKey != "key-b" {
		t.Error("RemoveCredentials() affected unrelated server entry")
	}
	// Default server should be cleared since it was removed
	if loaded.DefaultServer != "" {
		t.Errorf("DefaultServer = %q, want empty (was removed)", loaded.DefaultServer)
	}
}

func TestLoadCredentials_CorruptJSON(t *testing.T) {
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	_ = os.WriteFile(credsFile, []byte("not valid json{{{"), 0600)

	_, err := LoadCredentials(credsFile)
	if err == nil {
		t.Fatal("LoadCredentials() should error on corrupt JSON")
	}
}

func TestLoadCredentials_NilServersField(t *testing.T) {
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	// JSON with no "servers" key — Unmarshal leaves Servers as nil
	_ = os.WriteFile(credsFile, []byte(`{"default_server":"x"}`), 0600)

	creds, err := LoadCredentials(credsFile)
	if err != nil {
		t.Fatalf("LoadCredentials() error = %v", err)
	}
	if creds.Servers == nil {
		t.Fatal("LoadCredentials() should initialize nil Servers map")
	}
}

func TestGetAPIKey_NoDefaultServer(t *testing.T) {
	t.Setenv("DATADATDAT_API_KEY", "")
	if err := os.Unsetenv("DATADATDAT_API_KEY"); err != nil {
		t.Fatalf("failed to unset env: %v", err)
	}

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	creds := Credentials{
		Servers: map[string]ServerCredential{
			"http://localhost:8080": {APIKey: "key"},
		},
		DefaultServer: "", // no default
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	key := GetAPIKey(credsFile)
	if key != "" {
		t.Errorf("GetAPIKey() = %q, want empty (no default server)", key)
	}
}

func TestGetAPIKey_DefaultServerNotInMap(t *testing.T) {
	t.Setenv("DATADATDAT_API_KEY", "")
	if err := os.Unsetenv("DATADATDAT_API_KEY"); err != nil {
		t.Fatalf("failed to unset env: %v", err)
	}

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	creds := Credentials{
		Servers:       map[string]ServerCredential{},
		DefaultServer: "http://gone:8080",
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	key := GetAPIKey(credsFile)
	if key != "" {
		t.Errorf("GetAPIKey() = %q, want empty (default server not in map)", key)
	}
}

func TestGetAPIKey_CorruptFile(t *testing.T) {
	t.Setenv("DATADATDAT_API_KEY", "")
	if err := os.Unsetenv("DATADATDAT_API_KEY"); err != nil {
		t.Fatalf("failed to unset env: %v", err)
	}

	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	_ = os.WriteFile(credsFile, []byte("corrupt"), 0600)

	key := GetAPIKey(credsFile)
	if key != "" {
		t.Errorf("GetAPIKey() = %q, want empty (corrupt file)", key)
	}
}

func TestGetAPIKeyForServer_CorruptFile(t *testing.T) {
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	_ = os.WriteFile(credsFile, []byte("corrupt"), 0600)

	key := GetAPIKeyForServer(credsFile, "http://localhost:8080")
	if key != "" {
		t.Errorf("GetAPIKeyForServer() = %q, want empty (corrupt file)", key)
	}
}

func TestRemoveCredentials_CorruptFile(t *testing.T) {
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	_ = os.WriteFile(credsFile, []byte("corrupt"), 0600)

	err := RemoveCredentials(credsFile, "http://localhost:8080")
	if err == nil {
		t.Fatal("RemoveCredentials() should error on corrupt file")
	}
}

func TestRemoveCredentials_NonexistentServer(t *testing.T) {
	tmpDir := t.TempDir()
	credsFile := filepath.Join(tmpDir, "credentials")
	creds := Credentials{
		Servers:       map[string]ServerCredential{},
		DefaultServer: "",
	}
	data, _ := json.Marshal(creds)
	_ = os.WriteFile(credsFile, data, 0600)

	// Should not error on nonexistent server
	err := RemoveCredentials(credsFile, "http://nonexistent:8080")
	if err != nil {
		t.Fatalf("RemoveCredentials() error = %v, want nil for nonexistent", err)
	}
}
