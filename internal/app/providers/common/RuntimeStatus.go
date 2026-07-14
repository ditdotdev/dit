// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

type runtimeStatus struct {
	name   string
	status string
}

func RuntimeStatus(name string, status string) runtimeStatus {
	return runtimeStatus{
		name:   name,
		status: status,
	}
}
