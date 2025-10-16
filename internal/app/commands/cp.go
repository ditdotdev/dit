package commands

import (
	"github.com/spf13/cobra"
)

var (
	destination string
)

// cpCmd represents the cp command
var cpCmd = &cobra.Command{
	Use:   "cp [REPOSITORY]",
	Short: "Copy data into a repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		provider.Copy(repo, "local", source, destination)
	},
}

func init() {
	rootCmd.AddCommand(cpCmd)
	cpCmd.Flags().StringVarP(&source, "source", "s", "", "source location of the files on the local machine (required)")
	cpCmd.Flags().StringVarP(&destination, "destination", "d", "", "destination of the files inside of the container")
	_ = cpCmd.MarkFlagRequired("source")
}
