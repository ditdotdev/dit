// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"net/http"
	"strings"
	"testing"
)

func TestTagCommit_AddsTagsWithAndWithoutValue(t *testing.T) {
	updated := false
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"abc","properties":{"tags":{"existing":"keep"}}}`))
		case http.MethodPost:
			updated = true
			_, _ = w.Write([]byte(`{"id":"abc","properties":{}}`))
		}
	})

	_ = captureStdout(func() {
		TagCommit("repo", "abc", []string{"v1", "env=prod"}, port)
	})

	if !updated {
		t.Errorf("expected UpdateCommit to be called")
	}
}

func TestTagCommit_NoExistingTagsMap(t *testing.T) {
	updated := false
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"abc","properties":{}}`))
		case http.MethodPost:
			updated = true
			_, _ = w.Write([]byte(`{"id":"abc","properties":{}}`))
		}
	})

	_ = captureStdout(func() {
		TagCommit("repo", "abc", []string{"v1"}, port)
	})

	if !updated {
		t.Errorf("expected UpdateCommit to be called")
	}
}

func TestTagCommit_UpdateError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"abc","properties":{"tags":{}}}`))
		case http.MethodPost:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":"X","message":"boom","details":""}`))
		}
	})

	output := captureStdout(func() {
		TagCommit("repo", "abc", []string{"v1"}, port)
	})

	if !strings.Contains(output, "Error updating commit tags") {
		t.Errorf("expected error message, got %q", output)
	}
}
