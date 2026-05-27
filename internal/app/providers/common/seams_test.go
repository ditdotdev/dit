package common

import "testing"

func TestSetOsExitForTesting_RoundTrip(t *testing.T) {
	called := false
	prev := SetOsExitForTesting(func(int) { called = true })
	defer SetOsExitForTesting(prev)

	osExit(7)
	if !called {
		t.Error("SetOsExitForTesting did not install the swap")
	}
}
