package local

import (
	"context"
	"os"

	client "github.com/datadatdat/datadatdat-client-go"
)

func init() {
	_, d := os.LookupEnv("DATADATDAT_DEBUG")
	cfg.Debug = d
	// Enable command executor debug mode if DATADATDAT_DEBUG is set
	if d {
		ce.SetDebug(true)
	}
}

var cfg = client.NewConfiguration()
var apiClient = client.NewAPIClient(cfg)
var commitsApi = apiClient.CommitsAPI
var repositoriesApi = apiClient.RepositoriesAPI
var volumesApi = apiClient.VolumesAPI
var ctx = context.Background()
