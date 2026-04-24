package commands

import (
	"datadatdat/internal/app"
	"datadatdat/internal/app/providers"
	"fmt"
	"os"
	"os/user"

	"github.com/spf13/cobra"

	// Import remote providers to register them
	_ "github.com/datadatdat/datadatdat-remote-go/datadatdat"
	_ "github.com/datadatdat/nop-remote-go/nop"
	_ "github.com/datadatdat/s3-remote-go/s3"
	_ "github.com/datadatdat/s3web-remote-go/s3web"
	_ "github.com/datadatdat/ssh-remote-go/ssh"
)

var (
	context      string
	provider     providers.Provider
	version      string
	verbose      bool
	force        bool
	guid         string
	tags         []string
	params       []string
	envVars      []string
	name         string
	source       string
	remote       string
	updateOnly   bool
	removeImages bool
	privileged   bool
)

// Version will be set at build time via -ldflags
// This is an alias to app.DatadatdatVersion for backwards compatibility
var Version = app.DatadatdatVersion

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "d3",
	Short: "Datadatdat CLI",
	Long:  `Datadatdat CLI`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initProvider()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	//Global params
	rootCmd.PersistentFlags().StringVar(&context, "context", "", "Datadatdat Provider Context")
	rootCmd.Version = Version // Use dynamic version set at build time
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	u, _ := user.Current()
	datadatdatConfig := u.HomeDir + "/.datadatdat/config"
	if _, err := os.Stat(datadatdatConfig); os.IsNotExist(err) {
		// #nosec G304 -- Creating config file in user's home directory, path is controlled
		if _, err := os.Create(datadatdatConfig); err != nil {
			fmt.Printf("Error creating config file: %v\n", err)
		}
	}
}

// initProvider resolves the provider context. Called from rootCmd's PersistentPreRun
// (overridden by auth/org commands that don't need a provider).
//
// Looks up the context by exact name — it MUST NOT mutate the package-level
// `name` variable, which is bound to the `-n` flag for `run` and belongs to
// the repository name, not the context.
func initProvider() {
	isInstall := false
	for _, item := range os.Args {
		if item == "install" || item == "ls" {
			isInstall = true
		}
	}
	ctx := context
	if ctx == "" {
		ctx = os.Getenv("DATADATDAT_CONTEXT")
	}
	if ctx != "" {
		p, ok := providers.List()[ctx]
		if !ok {
			fmt.Fprintln(os.Stderr, "Error: no such context '"+ctx+"'")
			os.Exit(1)
		}
		provider = p
		context = ctx
		return
	}
	if isInstall {
		context = "docker" //TODO confirm valid
		return
	}
	provider = providers.Default()
}
