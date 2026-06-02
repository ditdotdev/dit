package commands

import (
	"github.com/ditdotdev/dit/internal/app/providers"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [IMAGE]",
	Short: "Create repository and start container",
	Long: `Create repository and start container.
Containers associated with a repository can be launched using context specific
run arguments and passed verbatim using '--' as the flag.

Docker example: 'dit run --disable-port-mapping postgres -- -p 2345:5432'
Privileged example: 'dit run --privileged icr.io/db2_community/db2:latest'`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		image := args[0]
		p, repoName := providers.Resolve(context, name)
		p.Run(image, repoName, envVars, args[1:], disablePortMap, privileged)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&name, nameKey, "n", "", "optional new name for repository")
	runCmd.Flags().StringSliceVarP(&envVars, "env", "e", nil, "container specific environment variables")
	runCmd.Flags().BoolVarP(&disablePortMap, "disable-port-mapping", "P", false, "disable default port mapping from container to localhost")
	runCmd.Flags().BoolVar(&privileged, "privileged", false, "run container in privileged mode with extended permissions")
	runCmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "filter latest commit by tags")
	runCmd.Flags().SortFlags = false //TODO review flag sorting
}
