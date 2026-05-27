package clients

import (
	"datadatdat/internal/app/utils"
	"os"
)

// commandRunner is the narrow interface around the part of
// utils.commandExecutor that production code uses. Existing as a var-bound
// indirection lets tests substitute an in-memory fake so the shell-out
// methods on `docker`/`kubernetes` are exercised without forking real
// subprocesses (which on dev workstations with Docker Desktop installed
// would take minutes per test run).
type commandRunner interface {
	Exec(name string, arg ...string) (string, error)
}

// realCE is the production-mode CommandExecutor; tests replace `ce` with
// a fake but must restore realCE on cleanup so other packages see the
// expected behavior.
var realCE = utils.CommandExecutor(60, false)
var ce commandRunner = realCE

func init() {
	_, d := os.LookupEnv("DATADATDAT_DEBUG")
	// Enable command executor debug mode if DATADATDAT_DEBUG is set
	if d {
		realCE.SetDebug(true)
		ce = realCE
	}
}
