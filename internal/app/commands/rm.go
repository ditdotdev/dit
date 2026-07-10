// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package commands

import (
	"github.com/ditdotdev/dit/internal/app/providers"
	"github.com/spf13/cobra"
)

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:   "rm [REPOSITORY]",
	Short: "Remove a repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		p, repo := providers.Resolve(context, args[0])
		p.Remove(repo, force)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&force, "force", "f", false, "Destroy all repositories")
}
