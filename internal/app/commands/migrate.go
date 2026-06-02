package commands

import (
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate [REPOSITORY]",
	Short: "Migrate an existing docker database container to dit repository",
	Long: `Migrate an existing docker database container to dit repository. 
Container becomes the new name of the docker container.

Example: 'dit migrate -s oldPostgres ditPostgres'`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider.Migrate(source, repo)
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().StringVarP(&source, "source", "s", "", "source docker database container (required)")
}
