// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package commands

import (
	"fmt"
	"github.com/ditdotdev/dit/internal/app/providers"
	"os"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "ls",
	Short: "List repositories",

	Run: func(cmd *cobra.Command, args []string) {
		h := fmt.Sprintf("%-12s  %-20s  %s", "CONTEXT", "REPOSITORY", "STATUS")
		fmt.Println(h)
		if context != "" {
			p, ok := providers.List()[context]
			if !ok {
				fmt.Fprintln(os.Stderr, "Error: no such context '"+context+"'")
				os.Exit(1)
			}
			p.List(context)
			return
		}
		for key, p := range providers.List() {
			p.List(key)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
