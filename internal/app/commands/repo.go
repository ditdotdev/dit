// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ditdotdev/dit/internal/app/providers/common"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   subcmdRepo,
	Short: "Manage server-side repositories",
	Long:  `Create, delete, and list repositories on a dit remote server.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Override root's PersistentPreRun to skip provider initialization.
		// Repo commands talk directly to the remote server, not the local dit provider.
	},
}

var repoCreateCmd = &cobra.Command{
	Use:   "create <org> <repo>",
	Short: "Create a repository on the server",
	Long:  `Create a new repository in the specified organization on a dit remote server.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		org := args[0]
		repo := args[1]
		server, _ := cmd.Flags().GetString(flagServer)
		private, _ := cmd.Flags().GetBool(flagPrivate)

		apiKey, server, err := resolveRepoAuth(server)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/api/v1/repos/%s/%s", server, org, repo)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)

		switch resp.StatusCode {
		case http.StatusCreated, http.StatusOK:
			cmd.Printf("Created repository %s/%s\n", org, repo)
		case http.StatusConflict:
			return fmt.Errorf("repository %s/%s already exists", org, repo)
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed (401); check your API key")
		default:
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
		}

		// --private requests an additional visibility update against the
		// auth-server endpoint (repos are public by default on creation).
		if private {
			if err := setRepoVisibility(apiKey, server, org, repo, true); err != nil {
				return fmt.Errorf("repository created but failed to set visibility: %w", err)
			}
			cmd.Printf("Set repository %s/%s visibility to private\n", org, repo)
		}
		return nil
	},
}

var repoSetVisibilityCmd = &cobra.Command{
	Use:   "set-visibility <org> <repo>",
	Short: "Set a repository's visibility (public or private)",
	Long:  `Change whether a repository is public or private on a dit remote server. Specify exactly one of --private or --public.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		org := args[0]
		repo := args[1]
		server, _ := cmd.Flags().GetString(flagServer)
		private, _ := cmd.Flags().GetBool(flagPrivate)
		public, _ := cmd.Flags().GetBool(flagPublic)

		if private == public {
			return fmt.Errorf("specify exactly one of --private or --public")
		}

		apiKey, server, err := resolveRepoAuth(server)
		if err != nil {
			return err
		}

		if err := setRepoVisibility(apiKey, server, org, repo, private); err != nil {
			return err
		}

		visibility := "public"
		if private {
			visibility = "private"
		}
		cmd.Printf("Set repository %s/%s visibility to %s\n", org, repo, visibility)
		return nil
	},
}

// setRepoVisibility PATCHes the auth-server visibility endpoint for a repo.
func setRepoVisibility(apiKey, server, org, repo string, private bool) error {
	bodyJSON, err := json.Marshal(map[string]bool{"isPrivate": private})
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/visibility", server, org, repo)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("repository %s/%s not found", org, repo)
	case http.StatusForbidden:
		return fmt.Errorf("forbidden (403); you do not have permission to change this repository's visibility")
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed (401); check your API key")
	default:
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}
}

var repoDeleteCmd = &cobra.Command{
	Use:   "delete <org> <repo>",
	Short: "Delete a repository from the server",
	Long:  `Delete a repository from the specified organization on a dit remote server.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		org := args[0]
		repo := args[1]
		server, _ := cmd.Flags().GetString("server")

		apiKey, server, err := resolveRepoAuth(server)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/api/v1/repos/%s/%s", server, org, repo)
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)

		switch resp.StatusCode {
		case http.StatusOK, http.StatusNoContent:
			cmd.Printf("Deleted repository %s/%s\n", org, repo)
			return nil
		case http.StatusNotFound:
			return fmt.Errorf("repository %s/%s not found", org, repo)
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed (401); check your API key")
		default:
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
		}
	},
}

var repoListCmd = &cobra.Command{
	Use:     subcmdList,
	Aliases: []string{"ls"},
	Short:   "List repositories on the server",
	Long:    `List all repositories on a dit remote server. Use --org to filter by organization.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")
		org, _ := cmd.Flags().GetString("org")

		apiKey, server, err := resolveRepoAuth(server)
		if err != nil {
			return err
		}

		url := server + "/api/v1/repos"
		if org != "" {
			url = fmt.Sprintf("%s/api/v1/repos/%s", server, org)
		}

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("authentication failed (401); check your API key")
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
		}

		var repos []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(repos) == 0 {
			cmd.Println("No repositories found.")
			return nil
		}

		for _, r := range repos {
			repoOrg, _ := r["org"].(string)
			repoName, _ := r["repo"].(string)
			cmd.Printf("%s/%s\n", repoOrg, repoName)
		}

		return nil
	},
}

var repoCommitsCmd = &cobra.Command{
	Use:   "commits <org> <repo>",
	Short: "List a repository's commits",
	Long:  `List the commits of a repository on a dit remote server, newest first.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		org := args[0]
		repo := args[1]
		server, _ := cmd.Flags().GetString(flagServer)
		outputFormat, _ := cmd.Flags().GetString(flagOutput)

		apiKey, server, err := resolveRepoAuth(server)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/api/v1/repos/%s/%s/commits", server, org, repo)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("authentication failed (401); check your API key")
		}
		if resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("forbidden (403); you do not have permission to read this repository")
		}
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("repository %s/%s not found", org, repo)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
		}

		// The commits endpoint returns a paginated object:
		// {"commits": [{"commitId": ..., "message": ..., "timestamp": ...}, ...], "total": N, ...}
		var payload struct {
			Commits []map[string]any `json:"commits"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// JSON output is the machine-readable form: the raw commits array.
		if outputFormat == "json" {
			encoded, err := json.Marshal(payload.Commits)
			if err != nil {
				return fmt.Errorf("failed to encode commits: %w", err)
			}
			cmd.Println(string(encoded))
			return nil
		}

		if len(payload.Commits) == 0 {
			cmd.Println("No commits found.")
			return nil
		}

		for _, c := range payload.Commits {
			commitID, _ := c["commitId"].(string)
			message, _ := c["message"].(string)
			if message != "" {
				cmd.Printf("%s\t%s\n", commitID, message)
			} else {
				cmd.Println(commitID)
			}
		}

		return nil
	},
}

// resolveRepoAuth resolves API key and server URL using the same pattern as org commands.
func resolveRepoAuth(server string) (string, string, error) {
	credsPath := getCredentialsPath()

	apiKey := common.GetAPIKey(credsPath, server)

	if apiKey == "" {
		return "", "", fmt.Errorf("not authenticated; run 'dit auth login' or set DIT_API_KEY")
	}

	if server == "" {
		creds, err := common.LoadCredentials(credsPath)
		if err == nil && creds.DefaultServer != "" {
			server = creds.DefaultServer
		}
	}
	if server == "" {
		return "", "", fmt.Errorf("no server configured; use --server or run 'dit auth login --server <url>'")
	}

	return apiKey, server, nil
}

func init() {
	rootCmd.AddCommand(repoCmd)

	repoCmd.AddCommand(repoCreateCmd)
	repoCreateCmd.Flags().String(flagServer, "", "Server URL")
	repoCreateCmd.Flags().Bool(flagPrivate, false, "Create the repository as private")

	repoCmd.AddCommand(repoDeleteCmd)
	repoDeleteCmd.Flags().String(flagServer, "", "Server URL")
	repoDeleteCmd.Flags().Bool("force", false, "Skip confirmation")

	repoCmd.AddCommand(repoListCmd)
	repoListCmd.Flags().String(flagServer, "", "Server URL")
	repoListCmd.Flags().String("org", "", "Filter by organization")

	repoCmd.AddCommand(repoCommitsCmd)
	repoCommitsCmd.Flags().String(flagServer, "", "Server URL")
	repoCommitsCmd.Flags().StringP(flagOutput, "o", "", "Output format: json for machine-readable output")

	repoCmd.AddCommand(repoSetVisibilityCmd)
	repoSetVisibilityCmd.Flags().String(flagServer, "", "Server URL")
	repoSetVisibilityCmd.Flags().Bool(flagPrivate, false, "Make the repository private")
	repoSetVisibilityCmd.Flags().Bool(flagPublic, false, "Make the repository public")

	addRepoCollaboratorCommands()
}
