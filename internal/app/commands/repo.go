package commands

import (
	"datadatdat/internal/app/providers/common"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   subcmdRepo,
	Short: "Manage server-side repositories",
	Long:  `Create, delete, and list repositories on a datadatdat remote server.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Override root's PersistentPreRun to skip provider initialization.
		// Repo commands talk directly to the remote server, not the local d3 provider.
	},
}

var repoCreateCmd = &cobra.Command{
	Use:   "create <org> <repo>",
	Short: "Create a repository on the server",
	Long:  `Create a new repository in the specified organization on a datadatdat remote server.`,
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
			return nil
		case http.StatusConflict:
			return fmt.Errorf("repository %s/%s already exists", org, repo)
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed (401); check your API key")
		default:
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
		}
	},
}

var repoDeleteCmd = &cobra.Command{
	Use:   "delete <org> <repo>",
	Short: "Delete a repository from the server",
	Long:  `Delete a repository from the specified organization on a datadatdat remote server.`,
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
	Long:    `List all repositories on a datadatdat remote server. Use --org to filter by organization.`,
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

// resolveRepoAuth resolves API key and server URL using the same pattern as org commands.
func resolveRepoAuth(server string) (string, string, error) {
	credsPath := getCredentialsPath()

	apiKey := os.Getenv("DATADATDAT_API_KEY")
	if apiKey == "" {
		if server != "" {
			apiKey = common.GetAPIKeyForServer(credsPath, server)
		} else {
			apiKey = common.GetAPIKey(credsPath)
		}
	}

	if apiKey == "" {
		return "", "", fmt.Errorf("not authenticated; run 'd3 auth login' or set DATADATDAT_API_KEY")
	}

	if server == "" {
		creds, err := common.LoadCredentials(credsPath)
		if err == nil && creds.DefaultServer != "" {
			server = creds.DefaultServer
		}
	}
	if server == "" {
		return "", "", fmt.Errorf("no server configured; use --server or run 'd3 auth login --server <url>'")
	}

	return apiKey, server, nil
}

func init() {
	rootCmd.AddCommand(repoCmd)

	repoCmd.AddCommand(repoCreateCmd)
	repoCreateCmd.Flags().String("server", "", "Server URL")

	repoCmd.AddCommand(repoDeleteCmd)
	repoDeleteCmd.Flags().String("server", "", "Server URL")
	repoDeleteCmd.Flags().Bool("force", false, "Skip confirmation")

	repoCmd.AddCommand(repoListCmd)
	repoListCmd.Flags().String("server", "", "Server URL")
	repoListCmd.Flags().String("org", "", "Filter by organization")
}
