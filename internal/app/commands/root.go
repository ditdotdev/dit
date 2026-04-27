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

// classifyCommand inspects os.Args for the d3 invocation and returns
// whether the command can run without a resolved provider, plus the
// default context name to use when one is needed (only `d3 install`
// uses this — it creates the first context).
//
// Pre-fix this logic scanned os.Args for "ls" or "install" anywhere in
// the slice, which made `d3 remote ls <repo>` falsely qualify as
// "provider-optional" and panic with SIGSEGV when remote.go tried to
// dereference the nil provider. The fix: only consider the position
// directly after the binary name (the top-level subcommand), and
// recognize `context` as a parent verb whose subcommands manage the
// providers map / config directly.
func classifyCommand(args []string) (providerOptional bool, defaultContextName string) {
	if len(args) < 2 {
		return false, ""
	}
	switch args[1] {
	case "install":
		return true, "docker"
	case "ls", "context":
		return true, ""
	}
	return false, ""
}

// initProvider resolves the provider context. Called from rootCmd's PersistentPreRun
// (overridden by auth/org commands that don't need a provider).
//
// Looks up the context by exact name — it MUST NOT mutate the package-level
// `name` variable, which is bound to the `-n` flag for `run` and belongs to
// the repository name, not the context.
func initProvider() {
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
	if optional, defaultCtx := classifyCommand(os.Args); optional {
		// First-time install may have no contexts yet; default the
		// context name without resolving a provider (providers.Default()
		// would panic). For `d3 ls` and `d3 context ...`, leave both
		// `context` and `provider` empty — the relevant command handlers
		// iterate providers.List() or otherwise don't need a resolved
		// provider.
		if defaultCtx != "" {
			context = defaultCtx
		}
		return
	}
	provider = providers.Default()
}
