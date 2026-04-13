package commands

import (
	"bytes"
	"datadatdat/internal/app/providers/common"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Manage organizations",
	Long:  `Manage organizations on a datadatdat remote server.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Override root's PersistentPreRun to skip provider initialization.
		// Org commands talk directly to the remote server, not the local d3 provider.
	},
}

var orgListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List organizations you belong to",
	Long:    `List all organizations that the authenticated user is a member of.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")

		apiKey, server, err := resolveOrgAuth(server)
		if err != nil {
			return err
		}

		url := server + "/api/v1/orgs"
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

		var orgs []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(orgs) == 0 {
			cmd.Println("No organizations found.")
			return nil
		}

		for _, org := range orgs {
			orgName, _ := org["name"].(string)
			cmd.Println(orgName)
		}

		return nil
	},
}

var orgCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create an organization",
	Long:  `Create a new organization on a datadatdat remote server.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orgName := args[0]
		server, _ := cmd.Flags().GetString("server")
		displayName, _ := cmd.Flags().GetString("display-name")

		apiKey, server, err := resolveOrgAuth(server)
		if err != nil {
			return err
		}

		reqBody := map[string]string{"name": orgName}
		if displayName != "" {
			reqBody["displayName"] = displayName
		}
		bodyJSON, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to build request: %w", err)
		}

		url := server + "/api/v1/orgs"
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyJSON))
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
		case http.StatusCreated, http.StatusOK:
			cmd.Printf("Created organization %s\n", orgName)
			return nil
		case http.StatusConflict:
			return fmt.Errorf("organization %s already exists", orgName)
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed (401); check your API key")
		default:
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
		}
	},
}

var orgInfoCmd = &cobra.Command{
	Use:   "info <org-name>",
	Short: "Show organization details",
	Long:  `Display details for an organization on a datadatdat remote server.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orgName := args[0]
		server, _ := cmd.Flags().GetString("server")

		apiKey, server, err := resolveOrgAuth(server)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/api/v1/orgs/%s", server, orgName)
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
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("organization %s not found", orgName)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
		}

		var org map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		name, _ := org["name"].(string)
		display, _ := org["displayName"].(string)
		cmd.Printf("Name:         %s\n", name)
		if display != "" {
			cmd.Printf("Display Name: %s\n", display)
		}

		return nil
	},
}

var orgMembersCmd = &cobra.Command{
	Use:   "members <org-name>",
	Short: "List organization members",
	Long:  `List all members of an organization on a datadatdat remote server.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orgName := args[0]
		server, _ := cmd.Flags().GetString("server")

		apiKey, server, err := resolveOrgAuth(server)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/api/v1/orgs/%s/members", server, orgName)
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
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("organization %s not found", orgName)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
		}

		var members []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(members) == 0 {
			cmd.Println("No members found.")
			return nil
		}

		for _, m := range members {
			username, _ := m["username"].(string)
			role, _ := m["role"].(string)
			cmd.Printf("%-20s  %s\n", username, role)
		}

		return nil
	},
}

// resolveOrgAuth resolves API key and server URL for org commands.
func resolveOrgAuth(server string) (string, string, error) {
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
	rootCmd.AddCommand(orgCmd)

	orgCmd.AddCommand(orgListCmd)
	orgListCmd.Flags().String("server", "", "Server URL")

	orgCmd.AddCommand(orgCreateCmd)
	orgCreateCmd.Flags().String("server", "", "Server URL")
	orgCreateCmd.Flags().String("display-name", "", "Display name for the organization")

	orgCmd.AddCommand(orgInfoCmd)
	orgInfoCmd.Flags().String("server", "", "Server URL")

	orgCmd.AddCommand(orgMembersCmd)
	orgMembersCmd.Flags().String("server", "", "Server URL")
}
