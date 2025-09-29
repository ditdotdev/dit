package local

import (
	"context"
	client "github.com/datadatdat/titan-client-go"
	"os"
)

func init() {
	_, d := os.LookupEnv("TITAN_DEBUG")
	cfg.Debug = d
	// Enable command executor debug mode if TITAN_DEBUG is set
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
