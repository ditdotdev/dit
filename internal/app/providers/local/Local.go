package local

import (
	"context"
	client "github.com/datadatdat/datadatdat-client-go"
	"os"
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
var commitsApi = apiClient.CommitsApi
var repositoriesApi = apiClient.RepositoriesApi
var volumesApi = apiClient.VolumesApi
var ctx = context.Background()
