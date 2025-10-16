package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	path     string
	finalize bool
)

// upgradeCmd represents the upgrade command
var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade datadatdat CLI and infrastructure",
	Run: func(cmd *cobra.Command, args []string) {
		provider.Upgrade(force, version, finalize, path)
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().BoolVarP(&force, "force", "f", false, "destroy all repositories")
	upgradeCmd.Flags().StringVarP(&path, "path", "p", "", "full installation path of Datadatdat")
	upgradeCmd.Flags().BoolVar(&force, "finalize", false, "")
	upgradeCmd.Flags().SortFlags = false
	if err := upgradeCmd.Flags().MarkHidden("finalize"); err != nil {
		fmt.Printf("Warning: Failed to mark finalize flag as hidden: %v\n", err)
	}
}
