// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// Regression for #207: a ContainerExists failure (docker daemon down, CLI
// missing) was swallowed and Run stumbled on into confusing downstream
// errors instead of stopping with the cause.
func TestRun_ContainerExistsErrorExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{containerExistsErr: errors.New("docker daemon unreachable")}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				_, _ = Run("img", "", nil, nil, false, false, false, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected exit 1 when ContainerExists fails, got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "docker daemon unreachable") {
		t.Errorf("expected the underlying error in output, got %q", output)
	}
}
