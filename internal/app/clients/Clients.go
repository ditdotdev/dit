package clients

import (
	"os"
	"titan/internal/app/utils"
)

var ce = utils.CommandExecutor(60, false)

func init() {
	_, d := os.LookupEnv("TITAN_DEBUG")
	// Enable command executor debug mode if TITAN_DEBUG is set
	if d {
		ce.SetDebug(true)
	}
}
