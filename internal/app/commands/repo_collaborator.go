package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var repoCollaboratorCmd = &cobra.Command{
	Use:     "collaborator",
	Aliases: []string{"collab"},
	Short:   "Manage repository collaborators",
	Long:    `Add, remove, and list collaborators on a dit remote repository.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Override root's PersistentPreRun to skip provider initialization.
		// Collaborator commands talk directly to the remote server.
	},
}

var repoCollaboratorAddCmd = &cobra.Command{
	Use:   "add <org> <repo> <userId>",
	Short: "Add a collaborator to a repository",
	Long:  `Grant a user access to a repository with a permission level (read, write, or admin).`,
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		org, repo, userID := args[0], args[1], args[2]
		server, _ := cmd.Flags().GetString(flagServer)
		permission, _ := cmd.Flags().GetString(flagPermission)

		apiKey, server, err := resolveRepoAuth(server)
		if err != nil {
			return err
		}

		reqBody := map[string]string{"userId": userID, "permission": permission}
		bodyJSON, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to build request: %w", err)
		}

		url := fmt.Sprintf("%s/api/v1/repos/%s/%s/collaborators", server, org, repo)
		resp, body, err := doRepoCollaboratorRequest(http.MethodPost, url, apiKey, bodyJSON)
		if err != nil {
			return err
		}

		switch resp.StatusCode {
		case http.StatusOK, http.StatusCreated:
			cmd.Printf("Added %s as collaborator on %s/%s (%s)\n", userID, org, repo, permission)
			return nil
		case http.StatusConflict:
			return fmt.Errorf("user %s is already a collaborator on %s/%s", userID, org, repo)
		case http.StatusNotFound:
			return fmt.Errorf("repository %s/%s or user %s not found", org, repo, userID)
		case http.StatusForbidden:
			return fmt.Errorf("forbidden (403); you do not have permission to manage collaborators")
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed (401); check your API key")
		default:
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
		}
	},
}

var repoCollaboratorRemoveCmd = &cobra.Command{
	Use:     "remove <org> <repo> <userId>",
	Aliases: []string{"rm"},
	Short:   "Remove a collaborator from a repository",
	Long:    `Revoke a user's collaborator access to a repository.`,
	Args:    cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		org, repo, userID := args[0], args[1], args[2]
		server, _ := cmd.Flags().GetString(flagServer)

		apiKey, server, err := resolveRepoAuth(server)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/api/v1/repos/%s/%s/collaborators/%s", server, org, repo, userID)
		resp, body, err := doRepoCollaboratorRequest(http.MethodDelete, url, apiKey, nil)
		if err != nil {
			return err
		}

		switch resp.StatusCode {
		case http.StatusOK, http.StatusNoContent:
			cmd.Printf("Removed %s as collaborator on %s/%s\n", userID, org, repo)
			return nil
		case http.StatusNotFound:
			return fmt.Errorf("collaborator %s not found on %s/%s", userID, org, repo)
		case http.StatusForbidden:
			return fmt.Errorf("forbidden (403); you do not have permission to manage collaborators")
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed (401); check your API key")
		default:
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
		}
	},
}

var repoCollaboratorListCmd = &cobra.Command{
	Use:     "list <org> <repo>",
	Aliases: []string{"ls"},
	Short:   "List collaborators on a repository",
	Long:    `List all collaborators and their permissions on a dit remote repository.`,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		org, repo := args[0], args[1]
		server, _ := cmd.Flags().GetString(flagServer)

		apiKey, server, err := resolveRepoAuth(server)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/api/v1/repos/%s/%s/collaborators", server, org, repo)
		resp, body, err := doRepoCollaboratorRequest(http.MethodGet, url, apiKey, nil)
		if err != nil {
			return err
		}

		switch resp.StatusCode {
		case http.StatusOK:
		case http.StatusNotFound:
			return fmt.Errorf("repository %s/%s not found", org, repo)
		case http.StatusForbidden:
			return fmt.Errorf("forbidden (403); you do not have permission to view collaborators")
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed (401); check your API key")
		default:
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
		}

		var collabs []map[string]interface{}
		if err := json.Unmarshal(body, &collabs); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(collabs) == 0 {
			cmd.Println("No collaborators found.")
			return nil
		}

		for _, c := range collabs {
			uid, _ := c["userId"].(string)
			perm, _ := c[flagPermission].(string)
			cmd.Printf("%-40s  %s\n", uid, perm)
		}
		return nil
	},
}

// doRepoCollaboratorRequest performs an authenticated HTTP request against a
// collaborator endpoint and returns the response, the read body, and any
// transport error. The caller is responsible for inspecting the status code.
func doRepoCollaboratorRequest(method, url, apiKey string, jsonBody []byte) (*http.Response, []byte, error) {
	var reader io.Reader
	if jsonBody != nil {
		reader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	if jsonBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	return resp, body, nil
}

func addRepoCollaboratorCommands() {
	repoCmd.AddCommand(repoCollaboratorCmd)

	repoCollaboratorCmd.AddCommand(repoCollaboratorAddCmd)
	repoCollaboratorAddCmd.Flags().String(flagServer, "", "Server URL")
	repoCollaboratorAddCmd.Flags().String(flagPermission, "read", "Permission level: read, write, or admin")

	repoCollaboratorCmd.AddCommand(repoCollaboratorRemoveCmd)
	repoCollaboratorRemoveCmd.Flags().String(flagServer, "", "Server URL")

	repoCollaboratorCmd.AddCommand(repoCollaboratorListCmd)
	repoCollaboratorListCmd.Flags().String(flagServer, "", "Server URL")
}
