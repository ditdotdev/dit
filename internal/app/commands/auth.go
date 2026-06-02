package commands

import (
	"github.com/ditdotdev/dit/internal/app/providers/common"
	"fmt"

	"github.com/spf13/cobra"
)

// credentialsPathOverride allows tests to inject a custom credentials path.
// When empty, CredentialsPath() is used.
var credentialsPathOverride string

func getCredentialsPath() string {
	if credentialsPathOverride != "" {
		return credentialsPathOverride
	}
	return common.CredentialsPath()
}

var authCmd = &cobra.Command{
	Use:   subcmdAuth,
	Short: "Manage authentication credentials",
	Long:  `Manage authentication credentials for dit remote servers.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Override root's PersistentPreRun to skip provider initialization.
		// Auth commands don't need a configured dit provider context.
	},
}

var authLoginCmd = &cobra.Command{
	Use:   subcmdLogin,
	Short: "Store API key for a remote server",
	Long:  `Store an API key for authenticating with a dit remote server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, _ := cmd.Flags().GetString("api-key")
		server, _ := cmd.Flags().GetString("server")

		if apiKey == "" {
			return fmt.Errorf("--api-key is required")
		}

		credsPath := getCredentialsPath()
		creds, err := common.LoadCredentials(credsPath)
		if err != nil {
			return fmt.Errorf("failed to load credentials: %w", err)
		}

		creds.Servers[server] = common.ServerCredential{APIKey: apiKey}
		creds.DefaultServer = server

		if err := common.SaveCredentials(credsPath, creds); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		cmd.Println("Logged in to", server)
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   subcmdStatus,
	Short: "Show authentication status",
	Long:  `Show whether you are authenticated and which server is configured.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		credsPath := getCredentialsPath()
		creds, err := common.LoadCredentials(credsPath)
		if err != nil {
			cmd.Println("not authenticated")
			return nil
		}

		if creds.DefaultServer == "" || len(creds.Servers) == 0 {
			cmd.Println("not authenticated")
			return nil
		}

		if _, ok := creds.Servers[creds.DefaultServer]; !ok {
			cmd.Println("not authenticated")
			return nil
		}

		cmd.Printf("authenticated to %s\n", creds.DefaultServer)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   subcmdLogout,
	Short: "Remove stored credentials for a server",
	Long:  `Remove stored API key credentials for a dit remote server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")
		credsPath := getCredentialsPath()

		if err := common.RemoveCredentials(credsPath, server); err != nil {
			return fmt.Errorf("failed to remove credentials: %w", err)
		}

		cmd.Println("Logged out from", server)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)

	authCmd.AddCommand(authLoginCmd)
	authLoginCmd.Flags().String("api-key", "", "API key for authentication (required)")
	authLoginCmd.Flags().String("server", "", "Server URL")

	authCmd.AddCommand(authStatusCmd)

	authCmd.AddCommand(authLogoutCmd)
	authLogoutCmd.Flags().String("server", "", "Server URL to logout from")
}
