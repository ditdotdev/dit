package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"datadatdat/internal/app/providers"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "ls",
	Short: "List repositories",

	Run: func(cmd *cobra.Command, args []string) {
		h := fmt.Sprintf("%-12s  %-20s  %s", "CONTEXT", "REPOSITORY", "STATUS")
		fmt.Println(h)
		for key, provider := range providers.List() {
			provider.List(key)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
