package kubernetes

import (
	"context"
	"datadatdat/internal/app/clients"
	"os"

	client "github.com/datadatdat/datadatdat-client-go"
)

func init() {
	_, d := os.LookupEnv("DATADATDAT_DEBUG")
	cfg.Debug = d
}

var cfg = client.NewConfiguration()
var apiClient = client.NewAPIClient(cfg)
var repositoriesApi = apiClient.RepositoriesAPI
var commitsApi = apiClient.CommitsAPI
var volumesApi = apiClient.VolumesAPI
var ctx = context.Background()

var k8s = clients.Kubernetes("default", "localhost", 5001)
