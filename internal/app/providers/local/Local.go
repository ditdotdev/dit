package local

import (
	"context"
	client "github.com/ditdotdev/dit-client-go"
	"os"
)

func init() {
	_, d := os.LookupEnv("DIT_DEBUG")
	cfg.Debug = d
	// Enable command executor debug mode if DIT_DEBUG is set
	if d {
		ce.SetDebug(true)
	}
}

var cfg = client.NewConfiguration()
var apiClient = client.NewAPIClient(cfg)
var commitsApi = apiClient.CommitsApi
var repositoriesApi = apiClient.RepositoriesApi
var volumesApi = apiClient.VolumesApi
var ctx = context.Background()

// stateRunning is the docker container State.Status value the package treats as
// "currently up." Extracted as a constant so goconst doesn't flag the literal
// across Remove / Copy / Migrate / tests.
const stateRunning = "running"

// flagName is the docker --name flag, extracted to satisfy goconst.
const flagName = "--name"
