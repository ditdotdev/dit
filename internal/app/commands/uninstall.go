package commands

import (
	"datadatdat/internal/app/providers"

	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   subcmdUninstall,
	Short: "Uninstall datadatdat infrastructure",

	Run: func(cmd *cobra.Command, args []string) {
		// `d3 uninstall --context X` targets only X. Without the flag, the
		// natural reading is "remove datadatdat from this machine," not
		// "remove the default context" — so iterate every configured
		// context. See #110: prior to this, a single `d3 uninstall` left
		// non-default contexts (their server containers, ZFS pools, data
		// volumes) behind, which also blocked image removal because the
		// orphaned containers held references to datadatdat:latest.
		if context != "" {
			provider.Uninstall(force, removeImages)
			providers.Remove(provider.GetName())
			return
		}

		// Snapshot the providers up front: providers.Remove mutates the
		// contexts map (and re-promotes a new default) on each call, so
		// iterating providers.List() directly would be unsafe.
		var snapshot []providers.Provider
		for _, p := range providers.List() {
			snapshot = append(snapshot, p)
		}
		for _, p := range snapshot {
			p.Uninstall(force, removeImages)
			providers.Remove(p.GetName())
		}
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().BoolVarP(&force, "force", "f", false, "destroy all repositories")
	uninstallCmd.Flags().BoolVar(&removeImages, "remove-images", false, "remove datadatdat docker images")
}
