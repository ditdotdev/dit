package kubernetes

import (
	"context"
	"datadatdat/internal/app/clients"
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

var k8s = clients.Kubernetes("default", "localhost", 5001)
