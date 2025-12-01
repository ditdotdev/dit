package commands

import (
	"datadatdat/internal/app/providers"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [IMAGE]",
	Short: "Create repository and start container",
	Long: `Create repository and start container.
Containers associated with a repository can be launched using context specific
run arguments and passed verbatim using '--' as the flag.

Docker example: 'd3 run --disable-port-mapping postgres -- -p 2345:5432'
Privileged example: 'd3 run --privileged icr.io/db2_community/db2:latest'`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		image := args[0]
		provider, name := providers.ByName(name)
		provider.Run(image, name, envVars, args[1:], disablePortMap, privileged)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&name, "name", "n", "", "optional new name for repository")
	runCmd.Flags().StringSliceVarP(&envVars, "env", "e", nil, "container specific environment variables")
	runCmd.Flags().BoolVarP(&disablePortMap, "disable-port-mapping", "P", false, "disable default port mapping from container to localhost")
	runCmd.Flags().BoolVar(&privileged, "privileged", false, "run container in privileged mode with extended permissions")
	runCmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "filter latest commit by tags")
	runCmd.Flags().SortFlags = false //TODO review flag sorting
}
