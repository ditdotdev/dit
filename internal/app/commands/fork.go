package commands

import (
	cmn "datadatdat/internal/app/providers/common"

	"github.com/spf13/cobra"
)

var (
	forkOrg  string
	forkName string
)

// forkCmd represents the fork command
var forkCmd = &cobra.Command{
	Use:   "fork <remote-url>",
	Short: "Fork a remote repository",
	Long:  "Fork a remote repository into your namespace or a specified organization",
	Args:  cobra.ExactArgs(1),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Override root's PersistentPreRun to skip provider initialization.
		// Fork talks directly to the remote server, not through a local provider.
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmn.Fork(args[0], forkOrg, forkName)
	},
}

func init() {
	rootCmd.AddCommand(forkCmd)
	forkCmd.Flags().StringVar(&forkOrg, "org", "", "Target organization/namespace for the fork")
	forkCmd.Flags().StringVar(&forkName, "name", "", "Custom name for the forked repository")
}
