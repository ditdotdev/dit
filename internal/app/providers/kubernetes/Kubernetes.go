package kubernetes

import (
	"context"
	client "github.com/datadatdat/datadatdat-client-go"
	"os"
)

func init() {
	_, d := os.LookupEnv("DATADATDAT_DEBUG")
	cfg.Debug = d
}

var cfg = client.NewConfiguration()
var apiClient = client.NewAPIClient(cfg)
var repositoriesApi = apiClient.RepositoriesApi
var commitsApi = apiClient.CommitsApi
var volumesApi = apiClient.VolumesApi
var ctx = context.Background()
