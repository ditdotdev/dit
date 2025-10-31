package common

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode"

	client "github.com/datadatdat/datadatdat-client-go"
)

func init() {
	_, d := os.LookupEnv("DATADATDAT_DEBUG")
	cfg.Debug = d
}

var cfg = client.NewConfiguration()
var apiClient = client.NewAPIClient(cfg)
var commitsApi = apiClient.CommitsAPI
var operationsApi = apiClient.OperationsAPI
var remotesApi = apiClient.RemotesAPI
var repositoriesApi = apiClient.RepositoriesAPI
var volumesApi = apiClient.VolumesAPI
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
