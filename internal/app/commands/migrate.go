// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

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

The source container must be stopped. The data under each volume the image
declares is copied into the new repository, whatever backs it (anonymous
volume, named volume, or bind mount). If the copy fails, the repository is
rolled back and removed, the source container is left unchanged, and the
command exits 1.

If the image declares no VOLUME, or its data lives somewhere other than the
declared paths, pass --volume with each absolute container path to copy. The
paths given replace the image's declared volumes and become the repository's
volumes in order (v0, v1, ...).

Examples:
  dit migrate -s oldPostgres ditPostgres
  dit migrate -s legacy-sqlite -v /app/data sqlite-repo`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider.Migrate(source, repo, volumes)
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().StringVarP(&source, "source", "s", "", "source docker database container (required)")
	migrateCmd.Flags().StringArrayVarP(&volumes, "volume", "v", nil, "absolute container path to copy instead of the image's declared volumes (repeatable)")
}
