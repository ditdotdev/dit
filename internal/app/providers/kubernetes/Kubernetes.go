package kubernetes

import (
	"context"
	client "github.com/ditdotdev/dit-client-go"
	"os"
)

func init() {
	_, d := os.LookupEnv("DIT_DEBUG")
	cfg.Debug = d
}

var cfg = client.NewConfiguration()
var apiClient = client.NewAPIClient(cfg)
var repositoriesApi = apiClient.RepositoriesApi
var commitsApi = apiClient.CommitsApi
var volumesApi = apiClient.VolumesApi
var ctx = context.Background()
