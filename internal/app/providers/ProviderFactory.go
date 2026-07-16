// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const (
	ProviderTypeDocker     = "docker"
	ProviderTypeKubernetes = "kubernetes"

	// defaultDockerRegistry is the Docker Hub namespace where the
	// official dit images live. Used as the per-provider fallback
	// registry when no override is specified.
	defaultDockerRegistry = "ditdotdev"

	// defaultHost is the loopback host every provider binds to. The
	// dit-server API is intentionally not reachable from off-box.
	defaultHost = "localhost"
)

/**
 * The provider factory is responsible for managing multiple providers (contexts to the user). We keep track of
 * providers in the ~/.dit/config file, which is a YAML file that contains a list of contexts and their
 * configuration. Each provider corresponds to an instance of 'dit-server' running somewhere (currently only
 * the user's laptop). The config file keeps track of:
 *
 *      - The context name
 *      - The context type (kubernetes or local)
 *      - The host (always localhost)
 *      - The port to connect to (defaults to 5001)
 *      - Default indicator
 *
 * Additional configuration, such as the provider type and provider-specific configuration, is stored within
 * the dit-server instance and accessible through the getContext() client method. When a context is created, it
 * can be given a type ("docker" or "kubernetes") as well as context-specific configuration.
 *
 * Each repository is associated with a particular context, and can be referred to as "context/repo", or just
 * "repo" for convenience (if there is only one known context, or no conflicts exists).
 */

var Providers map[string]Provider

type context struct {
	isDefault   bool
	host        string
	port        int
	contextType string
}

func loadContext(r interface{}) context {
	m := r.(map[string]interface{})
	return context{
		isDefault:   m["default"].(bool),
		host:        m["host"].(string),
		port:        m["port"].(int),
		contextType: m["type"].(string),
	}
}

func writeContext(c context) map[string]interface{} {
	m := make(map[string]interface{})
	m["default"] = c.isDefault
	m["host"] = c.host
	m["port"] = c.port
	m["type"] = c.contextType
	return m
}

func init() {
	home, _ := os.UserHomeDir()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(home + "/.dit")
	err := viper.ReadInConfig()
	if err != nil {
		// Likely config file does not exists, create one.
		_ = os.Mkdir(home+"/.dit", 0750)
		configPath := home + "/.dit/config"
		// #nosec G304 -- Creating config file in user's home directory, path is controlled
		if _, err := os.Create(configPath); err != nil {
			panic("failed to create config file: " + err.Error())
		}
	}
	Providers = make(map[string]Provider)
	contexts := viper.GetStringMap("contexts")
	for index, item := range contexts {
		context := loadContext(item)
		switch context.contextType {
		case ProviderTypeDocker:
			Providers[index] = Local(index, context.host, context.port)
		case ProviderTypeKubernetes:
			Providers[index] = Kubernetes(index, context.host, context.port)
		}
	}
}

func ByName(n string) (Provider, string) {
	var p Provider
	if !strings.Contains(n, "/") {
		p = Default()
	} else {
		s := strings.Split(n, "/")
		for k := range Providers {
			if k == s[0] {
				p = Providers[k]
			}
		}
		n = s[1]
	}
	if p == nil {
		panic("no such context '" + n + "'")
	}
	return p, n
}

// Resolve returns the provider and repository name for a command that accepts
// a repository argument. Two ways to specify which context the argument lives
// in are honored, in this order:
//
//  1. context/repo notation embedded in arg (e.g. "prod/my-repo"). This wins
//     even when contextFlag is also set, since it's explicitly scoped.
//  2. contextFlag, populated from the global --context flag.
//
// When neither is provided, the default context is used. Exits with a clear
// error message if a named context does not exist.
func Resolve(contextFlag, arg string) (Provider, string) {
	if strings.Contains(arg, "/") {
		s := strings.SplitN(arg, "/", 2)
		p, ok := Providers[s[0]]
		if !ok {
			fmt.Fprintln(os.Stderr, "Error: no such context '"+s[0]+"'")
			osExit(1)
		}
		return p, s[1]
	}
	if contextFlag != "" {
		p, ok := Providers[contextFlag]
		if !ok {
			fmt.Fprintln(os.Stderr, "Error: no such context '"+contextFlag+"'")
			osExit(1)
		}
		return p, arg
	}
	return Default(), arg
}

func List() map[string]Provider {
	return Providers
}

func GetAvailablePort() int {
	// Loopback-only bind for port discovery. Binding to ":0" (all interfaces)
	// triggers a Windows Defender Firewall prompt every time a freshly-built
	// binary runs — including each test binary, which has a new hash per
	// compile. The OS assigns the same free port regardless of which
	// interface we bind, so 127.0.0.1 is functionally equivalent and avoids
	// the prompt entirely. The listener is closed immediately; the returned
	// port is just a hint the caller binds for real later.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	p := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return p
}

func Create(name string, provider string, port int) Provider {
	if Providers[name] != nil {
		fmt.Println("Error: context '" + name + "' already exists. Run 'dit uninstall' first or use 'dit upgrade'.")
		osExit(1)
	}
	var p Provider
	switch provider {
	case ProviderTypeDocker:
		p = Local(name, defaultHost, port)
	case ProviderTypeKubernetes:
		p = Kubernetes(name, defaultHost, port)
	default:
		// Without this an unknown -t fell through, Create returned a nil
		// Provider, and the install command panicked calling Install on it.
		fmt.Println("Error: unknown context type '" + provider + "' (valid types: " + ProviderTypeDocker + ", " + ProviderTypeKubernetes + ")")
		osExit(1)
	}
	return p
}

func AddProvider(p Provider) {
	contexts := viper.GetStringMap("contexts")
	context := context{
		isDefault:   len(contexts) < 1,
		host:        defaultHost,
		port:        p.GetPort(),
		contextType: p.GetType(),
	}
	contexts[p.GetName()] = writeContext(context)
	viper.Set("contexts", contexts)
	err := viper.WriteConfig()
	if err != nil {
		panic(err)
	}
}

func Remove(n string) {
	contexts := viper.GetStringMap("contexts")
	current := loadContext(contexts[n])
	delete(contexts, n)
	// If we delete the default provider, just pick first one to be default
	if current.isDefault && len(contexts) > 0 {
		for k, c := range contexts {
			context := loadContext(c)
			context.isDefault = true
			contexts[k] = writeContext(context)
			break
		}
	}
	viper.Set("contexts", contexts)
	err := viper.WriteConfig()
	if err != nil {
		panic(err)
	}
}

func SetDefault(n string) {
	contexts := viper.GetStringMap("contexts")
	// Guard before mutating: the loop below un-defaults every context and
	// re-defaults only a key matching n, so an unknown name used to leave
	// the config with NO default context at all.
	if _, ok := contexts[n]; !ok {
		fmt.Println("Error: no such context '" + n + "'")
		osExit(1)
	}
	for k, c := range contexts {
		context := loadContext(c)
		context.isDefault = false
		if k == n {
			context.isDefault = true
		}
		contexts[k] = writeContext(context)
	}
	viper.Set("contexts", contexts)
	err := viper.WriteConfig()
	if err != nil {
		panic(err)
	}
}

func DefaultName() string {
	contexts := viper.GetStringMap("contexts")
	if len(contexts) == 0 {
		panic("No context is configured, run 'dit install' or 'dit context install' to configure dit")
	}
	var name string
	if len(contexts) == 1 {
		for k := range contexts {
			name = k
		}
	} else {
		for k, context := range contexts {
			c := loadContext(context)
			if c.isDefault {
				name = k
				break
			}
		}
	}
	if name == "" {
		panic("More than one context specified, but no default set")
	}
	return name
}

func Default() Provider {
	return Providers[DefaultName()]
}
