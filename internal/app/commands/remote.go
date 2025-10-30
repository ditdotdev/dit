package commands

import (
	"datadatdat/internal/app/providers"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// remoteCmd represents the remote command
var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Add, log, ls and rm remotes",
}

// remoteAddCmd represents the remoteAdd command
var remoteAddCmd = &cobra.Command{
	Use:   "add [URI] [REPOSITORY]",
	Short: "Set remote destination for a repository",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		uri := args[0]
		repo := args[1]
		fp := make(map[string]string)
		for _, p := range params {
			fmt.Println(p)
			s := strings.Split(p, "=")
			if len(s) != 2 {
				fmt.Println("Parameters must be in key=value format.")
				os.Exit(1)
			}
			fp[s[0]] = s[1]
		}
		provider.RemoteAdd(repo, uri, name, fp)
	},
}

// remoteListCmd represents the remoteList command
var remoteListCmd = &cobra.Command{
	Use:   "ls [REPOSITORY]",
	Short: "List remotes for a repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider = providers.Default()
		provider.RemoteList(repo)
	},
}

// remoteLogCmd represents the remoteLog command
var remoteLogCmd = &cobra.Command{
	Use:   "log [REPOSITORY]",
	Short: "Display log on remote",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider.RemoteLog(repo, remote, tags)
	},
}

// remoteRemoveCmd represents the remoteRemove command
var remoteRemoveCmd = &cobra.Command{
	Use:   "rm [REPOSITORY] [REMOTE]",
	Short: "Remove remote from a repository",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		remote := args[1]
		provider.RemoteRemove(repo, remote)
	},
}

func init() {
	rootCmd.AddCommand(remoteCmd)
	remoteCmd.AddCommand(remoteAddCmd)
	remoteCmd.AddCommand(remoteListCmd)
	remoteCmd.AddCommand(remoteLogCmd)
	remoteCmd.AddCommand(remoteRemoveCmd)

	remoteAddCmd.Flags().StringVarP(&name, "remote", "r", "", "name of the remote provider, defaults to origin")
	remoteAddCmd.Flags().StringSliceVarP(&params, "parameters", "p", nil, "provider specific parameters. key=value format")
	remoteAddCmd.Flags().SortFlags = false

	remoteListCmd.Flags().StringVarP(&remote, "remote", "r", "origin", "name of the remote provider, defaults to origin")
	remoteListCmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "tag to filter latest commit, if commit is not specified")
	remoteListCmd.Flags().SortFlags = false

	remoteLogCmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "tag to filter latest commit, if commit is not specified")
	remoteListCmd.Flags().SortFlags = false
}
