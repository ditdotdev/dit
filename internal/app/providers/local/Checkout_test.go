package local

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestCheckout_NoCommitsForTags(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, &fakeDocker{}, func() {
				Checkout("repo", "", []string{"v1"}, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "no matching commits found") {
		t.Errorf("expected no-matching message, got %q", output)
	}
}

func TestCheckout_TagsResolvesToFirstCommit(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/commits") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":"abc","properties":{}}]`))
		case strings.Contains(r.URL.Path, "/commits/abc/checkout"):
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			withDocker(t, &fakeDocker{}, func() {
				Checkout("repo", "", []string{"v1"}, port, "ctx")
			})
		})
	})

	if didExit {
		t.Errorf("happy tag-path should not exit; output=%q", output)
	}
	if !strings.Contains(output, "abc checked out") {
		t.Errorf("expected checked-out message, got %q", output)
	}
}

func TestCheckout_NoSourceCommit(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/status") {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, &fakeDocker{}, func() {
				Checkout("repo", "", nil, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "no commits present") {
		t.Errorf("expected no-commits message, got %q", output)
	}
}

func TestCheckout_TagsAndCommitExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, &fakeDocker{}, func() {
				Checkout("repo", "abc", []string{"v1"}, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "tags and commit cannot both be specified") {
		t.Errorf("expected tags+commit error, got %q", output)
	}
}

func TestCheckout_HappyPathWithGuid(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/commits/abc/checkout") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	d := &fakeDocker{}
	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Checkout("repo", "abc", nil, port, "ctx")
			})
		})
	})

	if didExit {
		t.Errorf("happy path should not exit")
	}
	if !strings.Contains(output, "abc checked out") {
		t.Errorf("expected checked-out message, got %q", output)
	}
	if d.StopCalls != 1 {
		t.Errorf("expected Stop call, got %d", d.StopCalls)
	}
	if d.StartCalls != 1 {
		t.Errorf("expected Start call, got %d", d.StartCalls)
	}
}

func TestCheckout_CheckoutCommitError(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/checkout") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":"X","message":"err","details":""}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, &fakeDocker{}, func() {
				Checkout("repo", "abc", nil, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Error checking out commit abc") {
		t.Errorf("expected checkout error, got %q", output)
	}
}

func TestCheckout_StartFailsExits(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{startErr: errors.New("can't start")}
	var didExit bool
	var code int
	output := captureStdout(func() {
		didExit, code = captureExit(t, func() {
			withDocker(t, d, func() {
				Checkout("repo", "abc", nil, port, "ctx")
			})
		})
	})

	if !didExit || code != 1 {
		t.Errorf("expected osExit(1); got didExit=%v code=%d", didExit, code)
	}
	if !strings.Contains(output, "Error starting container repo") {
		t.Errorf("expected start error, got %q", output)
	}
}

func TestCheckout_StopFailIsWarning(t *testing.T) {
	port := startMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	d := &fakeDocker{stopErr: errors.New("stop failed")}
	var didExit bool
	output := captureStdout(func() {
		didExit, _ = captureExit(t, func() {
			withDocker(t, d, func() {
				Checkout("repo", "abc", nil, port, "ctx")
			})
		})
	})

	if didExit {
		t.Errorf("Stop failure is a warning, not an exit")
	}
	if !strings.Contains(output, "Warning: Failed to stop container repo") {
		t.Errorf("expected warning, got %q", output)
	}
}
