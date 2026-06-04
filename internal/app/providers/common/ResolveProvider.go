package common

import (
	"fmt"

	"github.com/ditdotdev/remote-sdk-go/remote"
)

// ResolveProvider looks up a registered remote provider by name. If no provider has been registered for the given
// name, it returns a descriptive error naming the missing provider — surfacing typos at the call site rather than
// letting them turn into nil dereferences later in GetParameters/ToURL.
func ResolveProvider(name string) (remote.Remote, error) {
	rem, ok := remote.Get(name)
	if !ok {
		return nil, fmt.Errorf("unknown remote provider %q", name)
	}
	return rem, nil
}
