package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// Fork forks a remote repository into the specified namespace.
func Fork(uri string, org string, name string) {
	// Parse the remote URL to extract org/repo and server base.
	// URL format: http(s)://host:port/source-org/source-repo
	parsed, err := url.Parse(uri)
	if err != nil {
		fmt.Printf("Error: invalid remote URL: %s\n", err)
		return
	}

	// Extract source org/repo from path
	pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(pathParts) < 2 {
		fmt.Println("Error: remote URL must include org/repo (e.g., http://host/org/repo)")
		return
	}

	sourceOrg := pathParts[len(pathParts)-2]
	sourceRepo := pathParts[len(pathParts)-1]

	// Build server base URL
	serverBase := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)

	// Get API key for authentication
	apiKey := GetAPIKeyForServer(getCredsPath(), serverBase)
	if apiKey == "" {
		fmt.Printf("Error: not authenticated with %s. Run 'd3 auth login' first.\n", serverBase)
		return
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
		fmt.Printf("Error: failed to build request: %s\n", err)
		return
	}

	// Build fork URL
	forkURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/fork", serverBase, sourceOrg, sourceRepo)

	req, err := http.NewRequest("POST", forkURL, bytes.NewReader(bodyJSON))
	if err != nil {
		fmt.Printf("Error: failed to create request: %s\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req) //nolint:gosec // URL from configured remote properties
	if err != nil {
		fmt.Printf("Error: fork request failed: %s\n", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusCreated:
		var result map[string]interface{}
		if err := json.Unmarshal(respBody, &result); err == nil {
			fmt.Printf("Forked %s/%s to %s/%s\n", sourceOrg, sourceRepo, result["org"], result["repo"])
			if forkedFrom, ok := result["forkedFrom"]; ok {
				fmt.Printf("  Forked from: %s\n", forkedFrom)
			}
		} else {
			fmt.Println("Fork completed successfully")
		}
	case http.StatusConflict:
		fmt.Println("Error: repository already exists in target namespace")
	case http.StatusNotFound:
		fmt.Printf("Error: source repository %s/%s not found\n", sourceOrg, sourceRepo)
	default:
		fmt.Printf("Error: fork failed (HTTP %d): %s\n", resp.StatusCode, string(respBody))
	}
}
