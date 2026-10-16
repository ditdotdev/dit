// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package commands

import (
	cmn "github.com/ditdotdev/dit/internal/app/providers/common"

	"github.com/spf13/cobra"
)

var (
	disablePortMap bool
)

// cloneCmd represents the clone command
var cloneCmd = &cobra.Command{
	Use:     "clone URI [ARGS...]",
	Short:   "Clone a remote repository to local repository",
	Args:    cobra.MinimumNArgs(1),
	PreRunE: validatePullFlag,
	Run: func(cmd *cobra.Command, args []string) {
		uri := args[0]
		policy, _ := cmn.ParsePullPolicy(pull)
		provider.Clone(uri, name, guid, params, args[1:], disablePortMap, tags, policy)
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)
	cloneCmd.Flags().StringVarP(&name, nameKey, "n", "", "optional new name for repository")
	cloneCmd.Flags().StringVarP(&guid, "commit", "c", "", "commit to checkout")
	cloneCmd.Flags().StringSliceVarP(&params, "parameters", "p", nil, "provider specific parameters. key=value format")
	cloneCmd.Flags().BoolVarP(&disablePortMap, "disable-port-mapping", "P", false, "disable default port mapping from container to localhost")
	cloneCmd.Flags().StringVar(&pull, "pull", "missing", pullFlagHelp)
	cloneCmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "filter latest commit by tags")
}
