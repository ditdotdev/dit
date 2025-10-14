package clients

import (
	"os"
	"datadatdat/internal/app/utils"
)

var ce = utils.CommandExecutor(60, false)

func init() {
	_, d := os.LookupEnv("DATADATDAT_DEBUG")
	// Enable command executor debug mode if DATADATDAT_DEBUG is set
	if d {
		ce.SetDebug(true)
	}
}
