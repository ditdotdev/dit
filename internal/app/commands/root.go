package commands

import (
	"datadatdat/internal/app/providers"
	"fmt"
	"os"
	"os/user"

	"github.com/spf13/cobra"

	// Import remote providers to register them
	_ "github.com/datadatdat/datadatdat-remote-go/datadatdat"
	// TODO: Uncomment when these providers have their Go packages implemented
	// _ "github.com/datadatdat/nop-remote-go/nop"
	// _ "github.com/datadatdat/s3-remote-go/s3"
	// _ "github.com/datadatdat/s3web-remote-go/s3web"
	// _ "github.com/datadatdat/ssh-remote-go/ssh"
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
)

// Version will be set at build time via -ldflags
var Version = "dev"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "datadatdat",
	Short: "DataDatDat CLI",
	Long:  `DataDatDat CLI`,
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
	rootCmd.PersistentFlags().StringVar(&context, "context", "", "DataDatDat Provider Context")
	rootCmd.Version = Version // Use dynamic version set at build time
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	u, _ := user.Current()
	datadatdatConfig := u.HomeDir + "/.datadatdat/config"
	if _, err := os.Stat(datadatdatConfig); os.IsNotExist(err) {
		//nolint:gosec // G304: Creating config file in user's home directory, path is controlled
		if _, err := os.Create(datadatdatConfig); err != nil {
			fmt.Printf("Error creating config file: %v\n", err)
		}
	}
	isInstall := false
	for _, item := range os.Args {
		if item == "install" || item == "ls" {
			isInstall = true
		}
	}
	if context != "" {
		provider, name = providers.ByName(context)
	} else if os.Getenv("DATADATDAT_CONTEXT") != "" {
		provider, name = providers.ByName(os.Getenv("DATADATDAT_CONTEXT"))
	} else if isInstall {
		context = "docker" //TODO confirm valid
	} else {
		provider = providers.Default()
	}
}
