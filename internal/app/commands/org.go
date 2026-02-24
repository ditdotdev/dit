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

		credsPath := getCredentialsPath()

		// Resolve API key: env var > stored credentials
		apiKey := os.Getenv("DATADATDAT_API_KEY")
		if apiKey == "" {
			if server != "" {
				apiKey = common.GetAPIKeyForServer(credsPath, server)
			} else {
				apiKey = common.GetAPIKey(credsPath)
			}
		}

		if apiKey == "" {
			return fmt.Errorf("not authenticated; run 'd3 auth login' or set DATADATDAT_API_KEY")
		}

		// Resolve server URL from stored credentials if not provided
		if server == "" {
			creds, err := common.LoadCredentials(credsPath)
			if err == nil && creds.DefaultServer != "" {
				server = creds.DefaultServer
			}
		}
		if server == "" {
			return fmt.Errorf("no server configured; use --server or run 'd3 auth login --server <url>'")
		}

		// Call the orgs API
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

func init() {
	rootCmd.AddCommand(orgCmd)

	orgCmd.AddCommand(orgListCmd)
	orgListCmd.Flags().String("server", "", "Server URL")
}
