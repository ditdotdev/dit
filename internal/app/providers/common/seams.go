package common

import (
	"github.com/ditdotdev/dit/internal/app/clients"
	"os"
)

// dockerClient is the subset of clients.Docker methods used by providers/common.
// Exists so package functions can be unit-tested with a mock instead of a real
// docker daemon.
type dockerClient interface {
	GetValFromContainer(c string, key ...string) (string, error)
	InspectImage(image string) (string, error)
	Pull(image string) (string, error)
}

// newDocker is a var-bound factory so tests can substitute a mock dockerClient.
var newDocker = func(context string, port int) dockerClient {
	return clients.Docker(context, port)
}

// osExit indirects os.Exit so tests can capture the requested exit code instead
// of the test process actually exiting. Production callers behave exactly as
// before.
var osExit = os.Exit

// SetOsExitForTesting swaps the package-level osExit seam. Callers from
// outside this package (notably tests in github.com/ditdotdev/dit/internal/app/providers
// that drive the local + kubernetes delegators) can intercept osExit
// without having to live inside this package. Returns the previous value
// so callers can restore it on cleanup.
//
// This is the minimum production-code change required to test the
// providers/Local.go and providers/Kubernetes.go delegators end-to-end.
// The alternative — reimplementing every delegator in a wrapper layer
// just for testability — would balloon the surface area significantly.
func SetOsExitForTesting(fn func(int)) func(int) {
	prev := osExit
	osExit = fn
	return prev
}
