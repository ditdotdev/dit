// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParsePullPolicy(t *testing.T) {
	cases := []struct {
		in      string
		want    PullPolicy
		wantErr bool
	}{
		{"", PullMissing, false},
		{"missing", PullMissing, false},
		{"always", PullAlways, false},
		{"never", PullNever, false},
		{"sometimes", "", true},
		{"Always", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParsePullPolicy(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParsePullPolicy(%q) expected error", tc.in)
				}
				if !strings.Contains(err.Error(), "invalid --pull value") {
					t.Errorf("error %q should name the flag", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePullPolicy(%q) unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParsePullPolicy(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRepoDigests(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty", "", nil},
		{"null", "null", nil},
		{"empty list", "[]", nil},
		{"single", "[\"postgres@sha256:abc\"]\n", []string{"postgres@sha256:abc"}},
		{"two with whitespace", "[\n  \"a@sha256:1\",\n  \"b@sha256:2\"\n]", []string{"a@sha256:1", "b@sha256:2"}},
		{"compact", `["a@sha256:1","b@sha256:2"]`, []string{"a@sha256:1", "b@sha256:2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RepoDigests(tc.raw); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("RepoDigests(%q) = %#v, want %#v", tc.raw, got, tc.want)
			}
		})
	}
}

const (
	ensureRef    = "repo:tag"
	ensureDigest = "repo@sha256:aaaa"
	ensureOther  = "repo@sha256:bbbb"
)

func ensure(t *testing.T, d *fakeCloneDocker, stored string, policy PullPolicy) (output string, err error) {
	t.Helper()
	output = captureStdout(func() {
		err = EnsureImage(d, ensureRef, stored, policy)
	})
	return output, err
}

// ---- missing ----

// Issue #247: a present image with an unreachable registry must be used as-is.
func TestEnsureImage_MissingPresentNoPull(t *testing.T) {
	d := &fakeCloneDocker{pullErr: errors.New("manifest unknown")}
	_, err := ensure(t, d, "", PullMissing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("expected no pulls, got %v", d.pulledRefs)
	}
}

func TestEnsureImage_MissingAbsentPulls(t *testing.T) {
	d := &fakeCloneDocker{inspectErr: errors.New("no such image"), pullMakesPresent: true}
	_, err := ensure(t, d, "", PullMissing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureRef}) {
		t.Errorf("expected one pull by tag, got %v", d.pulledRefs)
	}
}

func TestEnsureImage_MissingAbsentPullFailsErrors(t *testing.T) {
	d := &fakeCloneDocker{inspectErr: errors.New("no such image"), pullErr: errors.New("registry down")}
	_, err := ensure(t, d, "", PullMissing)
	if err == nil {
		t.Fatal("expected error when image is absent and pull fails")
	}
	if !strings.Contains(err.Error(), "not found locally") {
		t.Errorf("error should say the image was not found locally, got %q", err)
	}
	if len(d.pulledRefs) != 1 {
		t.Errorf("expected exactly one pull attempt, got %v", d.pulledRefs)
	}
}

// ---- never ----

func TestEnsureImage_NeverPresentMatchNoWarning(t *testing.T) {
	d := &fakeCloneDocker{repoDigests: `["` + ensureDigest + `"]`}
	out, err := ensure(t, d, ensureDigest, PullNever)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "differs from the digest") {
		t.Errorf("no warning expected on digest match, got %q", out)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("expected no pulls, got %v", d.pulledRefs)
	}
}

func TestEnsureImage_NeverPresentMismatchWarns(t *testing.T) {
	d := &fakeCloneDocker{repoDigests: `["` + ensureOther + `"]`}
	out, err := ensure(t, d, ensureDigest, PullNever)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "differs from the digest recorded") {
		t.Errorf("expected mismatch warning, got %q", out)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("--pull never must not pull, got %v", d.pulledRefs)
	}
}

func TestEnsureImage_NeverPresentUnknownDigestNoWarning(t *testing.T) {
	for _, stored := range []string{"", "<nil>"} {
		d := &fakeCloneDocker{repoDigests: `["` + ensureOther + `"]`}
		out, err := ensure(t, d, stored, PullNever)
		if err != nil {
			t.Fatalf("stored=%q unexpected error: %v", stored, err)
		}
		if strings.Contains(out, "differs from the digest") {
			t.Errorf("stored=%q: no warning expected when the commit has no digest, got %q", stored, out)
		}
	}
}

func TestEnsureImage_NeverAbsentErrors(t *testing.T) {
	d := &fakeCloneDocker{inspectErr: errors.New("no such image")}
	_, err := ensure(t, d, ensureDigest, PullNever)
	if err == nil {
		t.Fatal("expected error when image is absent under --pull never")
	}
	if !strings.Contains(err.Error(), "not present locally") || !strings.Contains(err.Error(), "--pull never") {
		t.Errorf("error should explain the policy, got %q", err)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("--pull never must not pull, got %v", d.pulledRefs)
	}
}

// ---- always ----

func TestEnsureImage_AlwaysDigestMatchSkipsPull(t *testing.T) {
	d := &fakeCloneDocker{repoDigests: `["` + ensureDigest + `"]`}
	_, err := ensure(t, d, ensureDigest, PullAlways)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("expected no pulls on digest match, got %v", d.pulledRefs)
	}
}

func TestEnsureImage_AlwaysDigestMatchSecondEntry(t *testing.T) {
	d := &fakeCloneDocker{repoDigests: `["` + ensureOther + `", "` + ensureDigest + `"]`}
	_, err := ensure(t, d, ensureDigest, PullAlways)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("expected no pulls when any RepoDigests entry matches, got %v", d.pulledRefs)
	}
}

func TestEnsureImage_AlwaysMismatchPullsByDigestAndTags(t *testing.T) {
	d := &fakeCloneDocker{repoDigests: `["` + ensureOther + `"]`}
	_, err := ensure(t, d, ensureDigest, PullAlways)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest}) {
		t.Errorf("expected a single pull by digest, got %v", d.pulledRefs)
	}
	if !reflect.DeepEqual(d.tagCalls, [][2]string{{ensureDigest, ensureRef}}) {
		t.Errorf("expected docker tag digest -> ref, got %v", d.tagCalls)
	}
}

func TestEnsureImage_AlwaysNoStoredDigestPullsTag(t *testing.T) {
	for _, stored := range []string{"", "<nil>"} {
		d := &fakeCloneDocker{repoDigests: `["` + ensureOther + `"]`}
		_, err := ensure(t, d, stored, PullAlways)
		if err != nil {
			t.Fatalf("stored=%q unexpected error: %v", stored, err)
		}
		if !reflect.DeepEqual(d.pulledRefs, []string{ensureRef}) {
			t.Errorf("stored=%q: expected a single pull by tag, got %v", stored, d.pulledRefs)
		}
		if len(d.tagCalls) != 0 {
			t.Errorf("stored=%q: no retag expected after a tag pull, got %v", stored, d.tagCalls)
		}
	}
}

func TestEnsureImage_AlwaysLocalUntaggedPullsByDigest(t *testing.T) {
	// A locally built image has no RepoDigests; the commit does. Pull by digest.
	d := &fakeCloneDocker{repoDigests: "[]"}
	_, err := ensure(t, d, ensureDigest, PullAlways)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest}) {
		t.Errorf("expected a single pull by digest, got %v", d.pulledRefs)
	}
}

func TestEnsureImage_AlwaysDigestPullFailsFallsBackToTag(t *testing.T) {
	d := &fakeCloneDocker{repoDigests: `["` + ensureOther + `"]`, pullErrFor: map[string]error{ensureDigest: errors.New("manifest unknown")}}
	out, err := ensure(t, d, ensureDigest, PullAlways)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest, ensureRef}) {
		t.Errorf("expected digest pull then tag pull, got %v", d.pulledRefs)
	}
	if len(d.tagCalls) != 0 {
		t.Errorf("no retag expected when the digest pull failed, got %v", d.tagCalls)
	}
	if strings.Contains(out, "Warning: unable to pull") {
		t.Errorf("tag pull succeeded, no warning expected, got %q", out)
	}
}

func TestEnsureImage_AlwaysPullFailsWithLocalWarns(t *testing.T) {
	d := &fakeCloneDocker{repoDigests: "[]", pullErr: errors.New("manifest unknown")}
	out, err := ensure(t, d, "", PullAlways)
	if err != nil {
		t.Fatalf("a present image must survive a failed refresh, got %v", err)
	}
	if !strings.Contains(out, "Warning: unable to pull") {
		t.Errorf("expected pull warning, got %q", out)
	}
}

func TestEnsureImage_AlwaysPullFailsNoLocalErrors(t *testing.T) {
	d := &fakeCloneDocker{inspectErr: errors.New("no such image"), pullErr: errors.New("registry down")}
	_, err := ensure(t, d, ensureDigest, PullAlways)
	if err == nil {
		t.Fatal("expected error when image is absent and every pull fails")
	}
	if !strings.Contains(err.Error(), "unable to pull") {
		t.Errorf("error should mention the failed pull, got %q", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest, ensureRef}) {
		t.Errorf("expected digest pull then tag pull, got %v", d.pulledRefs)
	}
}

func TestEnsureImage_AlwaysAbsentPullsAndSucceeds(t *testing.T) {
	d := &fakeCloneDocker{inspectErr: errors.New("no such image"), pullMakesPresent: true}
	_, err := ensure(t, d, ensureDigest, PullAlways)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest}) {
		t.Errorf("expected a single pull by digest, got %v", d.pulledRefs)
	}
	if len(d.tagCalls) != 1 {
		t.Errorf("expected the digest to be tagged as %s, got %v", ensureRef, d.tagCalls)
	}
}
