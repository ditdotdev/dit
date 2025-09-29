package common

import (
	"context"
	"fmt"
	client "github.com/datadatdat/titan-client-go"
	"os"
	"strings"
	"unicode"
)

func init() {
	_, d := os.LookupEnv("TITAN_DEBUG")
	cfg.Debug = d
}

var cfg = client.NewConfiguration()
var apiClient = client.NewAPIClient(cfg)
var commitsApi = apiClient.CommitsApi
var operationsApi = apiClient.OperationsApi
var remotesApi = apiClient.RemotesApi
var repositoriesApi = apiClient.RepositoriesApi
var volumesApi = apiClient.VolumesApi
var ctx = context.Background()

func ifContainsPrint(m map[string]interface{}, k string) {
	v, ok := m[k]
	if ok {
		// Replace deprecated strings.Title with manual title case
		title := strings.ToUpper(string(unicode.ToUpper(rune(k[0])))) + k[1:]
		out := fmt.Sprintf("%v: %v", title, v)
		fmt.Println(out)
	}
}
