package commands

import (
	"fmt"
	"github.com/ditdotdev/dit/internal/app/providers"
	"github.com/spf13/cobra"
	"os"
)

var contextType string
var contextName string

// contextCmd represents the context command
var contextCmd = &cobra.Command{
	Use:   subcmdContext,
	Short: "Manage dit contexts",
}

// contextInstallCmd represents the contextInstall command
var contextInstallCmd = &cobra.Command{
	Use:   subcmdInstall,
	Short: "Install a new context",
	Run: func(cmd *cobra.Command, args []string) {
		provider = providers.Create(contextName, contextType, providers.GetAvailablePort())
		provider.Install(params, verbose)
		providers.AddProvider(provider)
	},
}

// contextUninstallCmd represents the contextUninstall command
var contextUninstallCmd = &cobra.Command{
	Use:   "uninstall [CONTEXTNAME]",
	Short: "Uninstall a context",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contextName := args[0]
		p, ok := providers.List()[contextName]
		if !ok {
			fmt.Fprintln(os.Stderr, "Error: no such context '"+contextName+"'")
			os.Exit(1)
		}
		p.Uninstall(force, false)
		providers.Remove(contextName)
	},
}

// contextDefaultCmd represents the contextDefault command
var contextDefaultCmd = &cobra.Command{
	Use:   "default [CONTEXTNAME]",
	Short: "Get or set default context",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			p := providers.Default()
			fmt.Println(p.GetName())
			os.Exit(0)
		}
		context = args[0]
		providers.SetDefault(context)
	},
}

// contextListCmd represents the contextList command
var contextListCmd = &cobra.Command{
	Use:   "ls",
	Short: "List available contexts",
	Run: func(cmd *cobra.Command, args []string) {
		h := fmt.Sprintf("%-20s  %-12s", "NAME", "TYPE")
		fmt.Println(h)
		plist := providers.List()
		if len(plist) > 0 {
			defaultName := providers.DefaultName()
			for context, provider := range plist {
				if context == defaultName {
					context = context + " (*)"
				}
				l := fmt.Sprintf("%-20s  %-12s", context, provider.GetType())
				fmt.Println(l)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(contextInstallCmd)
	contextCmd.AddCommand(contextUninstallCmd)
	contextCmd.AddCommand(contextDefaultCmd)
	contextCmd.AddCommand(contextListCmd)

	contextInstallCmd.Flags().StringVarP(&contextType, "type", "t", "docker", "context type (docker or kubernetes)")
	contextInstallCmd.Flags().StringVarP(&contextName, nameKey, "n", "docker", "context name, defaults to context type")
	contextInstallCmd.Flags().StringSliceVarP(&params, "parameters", "p", nil, "context specific parameters. key=value format")
	contextInstallCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging")
	contextInstallCmd.Flags().SortFlags = false //TODO review flag sorting

	contextUninstallCmd.Flags().BoolVarP(&force, "force", "f", false, "destroy all repositories")
}
