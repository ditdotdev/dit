// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package commands

import (
	"github.com/spf13/cobra"
)

var (
	disablePortMap bool
)

// cloneCmd represents the clone command
var cloneCmd = &cobra.Command{
	Use:   "clone [URI]", //TODO format usage for variadic args
	Short: "Clone a remote repository to local repository",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		uri := args[0]
		provider.Clone(uri, name, guid, params, args[1:], disablePortMap, tags)
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)
	cloneCmd.Flags().StringVarP(&name, nameKey, "n", "", "optional new name for repository")
	cloneCmd.Flags().StringVarP(&guid, "commit", "c", "", "commit to checkout")
	cloneCmd.Flags().StringSliceVarP(&params, "parameters", "p", nil, "provider specific parameters. key=value format")
	cloneCmd.Flags().BoolVarP(&disablePortMap, "disable-port-mapping", "P", false, "disable default port mapping from container to localhost")
	cloneCmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "filter latest commit by tags")
}
