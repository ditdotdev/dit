package common

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
)

// ServerCredential holds authentication credentials for a single server.
type ServerCredential struct {
	APIKey string `json:"api_key"` // #nosec G117 -- not a hardcoded credential, just a struct field name
}

// Credentials holds all stored server credentials.
type Credentials struct {
	Servers       map[string]ServerCredential `json:"servers"`
	DefaultServer string                      `json:"default_server,omitempty"`
}

// CredentialsPath returns the default path for the credentials file.
func CredentialsPath() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return filepath.Join(u.HomeDir, ".datadatdat", "credentials")
}

// LoadCredentials reads credentials from the given file path.
// Returns empty credentials (not an error) if the file does not exist.
func LoadCredentials(path string) (Credentials, error) {
	empty := Credentials{Servers: make(map[string]ServerCredential)}

	data, err := os.ReadFile(path) // #nosec G304 -- path is from controlled config, not user input
	if err != nil {
		if os.IsNotExist(err) {
			return empty, nil
		}
		return empty, err
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return empty, err
	}
	if creds.Servers == nil {
		creds.Servers = make(map[string]ServerCredential)
	}
	return creds, nil
}

// SaveCredentials writes credentials to the given file path.
// Creates parent directories if needed. File is written with 0600 permissions.
func SaveCredentials(path string, creds Credentials) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// GetAPIKey returns the API key to use, checking:
// 1. DATADATDAT_API_KEY environment variable (highest priority)
// 2. Stored credentials for the default server
func GetAPIKey(credsPath string) string {
	if envKey := os.Getenv("DATADATDAT_API_KEY"); envKey != "" {
		return envKey
	}

	creds, err := LoadCredentials(credsPath)
	if err != nil {
		return ""
	}

	if creds.DefaultServer == "" {
		return ""
	}

	if server, ok := creds.Servers[creds.DefaultServer]; ok {
		return server.APIKey
	}

	return ""
}

// GetAPIKeyForServer returns the API key for a specific server.
func GetAPIKeyForServer(credsPath string, serverURL string) string {
	creds, err := LoadCredentials(credsPath)
	if err != nil {
		return ""
	}

	if server, ok := creds.Servers[serverURL]; ok {
		return server.APIKey
	}

	return ""
}

// RemoveCredentials removes credentials for a specific server.
func RemoveCredentials(credsPath string, serverURL string) error {
	creds, err := LoadCredentials(credsPath)
	if err != nil {
		return err
	}

	delete(creds.Servers, serverURL)

	// Clear default if it was the removed server
	if creds.DefaultServer == serverURL {
		creds.DefaultServer = ""
	}

	return SaveCredentials(credsPath, creds)
}
