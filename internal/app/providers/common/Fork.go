package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// credentialsPathOverride allows tests to inject a custom credentials path.
// When empty, CredentialsPath() is used.
var credentialsPathOverride string

func getCredsPath() string {
	if credentialsPathOverride != "" {
		return credentialsPathOverride
	}
	return CredentialsPath()
}

// Fork forks a remote repository into the specified namespace. It returns an
// error on any failure so the CLI exits non-zero (and callers can script it).
func Fork(uri string, org string, name string) error {
	// Parse the remote URL to extract org/repo and server base.
	// URL format: http(s)://host:port/source-org/source-repo
	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid remote URL: %w", err)
	}

	// Extract source org/repo from path
	pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(pathParts) < 2 {
		return fmt.Errorf("remote URL must include org/repo (e.g., http://host/org/repo)")
	}

	sourceOrg := pathParts[len(pathParts)-2]
	sourceRepo := pathParts[len(pathParts)-1]

	// Build server base URL
	serverBase := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)

	// Allow overriding the host-visible gateway URL for the request target. Needed
	// in environments where the remote URL uses a Docker-internal hostname (e.g.
	// dit-api-gateway) that isn't resolvable from the host process running dit.
	// Mirrors push/pull/clone (see ensureRemoteRepoExists in Push.go). Credentials
	// stay keyed on the original server; only the request target is rewritten.
	apiBase := serverBase
	if hostGateway := os.Getenv("DIT_HOST_GATEWAY"); hostGateway != "" {
		apiBase = hostGateway
	}

	// Get API key for authentication (env var > stored credentials for this server).
	apiKey := GetAPIKey(getCredsPath(), serverBase)
	if apiKey == "" {
		return fmt.Errorf("not authenticated with %s; run 'dit auth login' first", serverBase)
	}

	// Build request body
	body := map[string]string{
		"targetNamespace": org,
	}
	if name != "" {
		body["targetName"] = name
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	// Build fork URL (using the possibly host-overridden base).
	forkURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/fork", apiBase, sourceOrg, sourceRepo)

	req, err := http.NewRequest("POST", forkURL, bytes.NewReader(bodyJSON)) // #nosec G704 -- fork target is the user-supplied remote URL (+ optional DIT_HOST_GATEWAY), not untrusted server input
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req) // #nosec G704 -- URL from configured remote properties
	if err != nil {
		return fmt.Errorf("fork request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusCreated:
		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err == nil {
			fmt.Printf("Forked %s/%s to %s/%s\n", sourceOrg, sourceRepo, result["org"], result["repo"])
			if forkedFrom, ok := result["forkedFrom"]; ok {
				fmt.Printf("  Forked from: %s\n", forkedFrom)
			}
		} else {
			fmt.Println("Fork completed successfully")
		}
		return nil
	case http.StatusConflict:
		return fmt.Errorf("repository already exists in target namespace")
	case http.StatusNotFound:
		return fmt.Errorf("source repository %s/%s not found", sourceOrg, sourceRepo)
	default:
		return fmt.Errorf("fork failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}
}
