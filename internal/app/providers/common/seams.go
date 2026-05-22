package common

import (
	"datadatdat/internal/app/clients"
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
