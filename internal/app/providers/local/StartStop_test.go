package local

import (
	"errors"
	"strings"
	"testing"
)

func TestStart_Success(t *testing.T) {
	d := &fakeDocker{}
	var err error
	output := captureStdout(func() {
		withDocker(t, d, func() {
			err = Start("repo", 9999)
		})
	})
	if err != nil {
		t.Errorf("expected nil err, got %v", err)
	}
	if d.StartCalls != 1 {
		t.Errorf("expected Start call, got %d", d.StartCalls)
	}
	if !strings.Contains(output, "repo started") {
		t.Errorf("expected start message, got %q", output)
	}
}

func TestStart_DockerError(t *testing.T) {
	d := &fakeDocker{startErr: errors.New("no such container")}
	var err error
	output := captureStdout(func() {
		withDocker(t, d, func() {
			err = Start("repo", 9999)
		})
	})
	if err == nil {
		t.Errorf("expected error")
	}
	if !strings.Contains(output, "Error starting container repo") {
		t.Errorf("expected error message, got %q", output)
	}
}

func TestStop_Success(t *testing.T) {
	d := &fakeDocker{}
	var err error
	output := captureStdout(func() {
		withDocker(t, d, func() {
			err = Stop("repo", 9999)
		})
	})
	if err != nil {
		t.Errorf("expected nil err, got %v", err)
	}
	if d.StopCalls != 1 {
		t.Errorf("expected Stop call, got %d", d.StopCalls)
	}
	if !strings.Contains(output, "repo stopped") {
		t.Errorf("expected stop message, got %q", output)
	}
}

func TestStop_DockerError(t *testing.T) {
	d := &fakeDocker{stopErr: errors.New("no such container")}
	var err error
	output := captureStdout(func() {
		withDocker(t, d, func() {
			err = Stop("repo", 9999)
		})
	})
	if err == nil {
		t.Errorf("expected error")
	}
	if !strings.Contains(output, "Error stopping container repo") {
		t.Errorf("expected error message, got %q", output)
	}
}
