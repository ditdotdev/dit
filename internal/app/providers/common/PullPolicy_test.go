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
	// The reference EnsureImage hands back when the committed digest is used:
	// docker accepts name:tag@digest and resolves it by the digest.
	ensureRunRef = "repo:tag@sha256:aaaa"
)

func ensure(t *testing.T, d *fakeCloneDocker, stored string, policy PullPolicy) (ref string, output string, err error) {
	t.Helper()
	output = captureStdout(func() {
		ref, err = EnsureImage(d, ensureRef, stored, policy)
	})
	return ref, output, err
}

// presentWithDigest is a daemon that already holds the committed digest
// under image:tag.
func presentWithDigest() *fakeCloneDocker {
	return &fakeCloneDocker{repoDigests: `["` + ensureDigest + `"]`}
}

// presentMismatch is a daemon whose image:tag is some other image and which
// does not hold the committed digest at all.
func presentMismatch() *fakeCloneDocker {
	return &fakeCloneDocker{repoDigests: `["` + ensureOther + `"]`, absent: map[string]bool{ensureDigest: true}}
}

// absentAll is a daemon with neither image:tag nor the digest.
func absentAll() *fakeCloneDocker {
	return &fakeCloneDocker{inspectErr: errors.New("no such image")}
}

// ---- missing ----

// Issue #247: a present image with an unreachable registry must be used as-is.
func TestEnsureImage_MissingPresentNoPull(t *testing.T) {
	d := &fakeCloneDocker{pullErr: errors.New("manifest unknown")}
	ref, _, err := ensure(t, d, "", PullMissing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("expected no pulls, got %v", d.pulledRefs)
	}
	if ref != ensureRef {
		t.Errorf("expected run ref %s, got %s", ensureRef, ref)
	}
}

func TestEnsureImage_MissingAbsentPulls(t *testing.T) {
	d := absentAll()
	d.pullMakesPresent = true
	ref, _, err := ensure(t, d, "", PullMissing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureRef}) {
		t.Errorf("expected one pull by tag, got %v", d.pulledRefs)
	}
	if ref != ensureRef {
		t.Errorf("expected run ref %s, got %s", ensureRef, ref)
	}
}

func TestEnsureImage_MissingAbsentPullFailsErrors(t *testing.T) {
	d := absentAll()
	d.pullErr = errors.New("registry down")
	_, _, err := ensure(t, d, "", PullMissing)
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

// Issue #240: the default policy runs the digest recorded in the commit.

func TestEnsureImage_MissingPresentDigestMatchRunsDigestNoPull(t *testing.T) {
	d := presentWithDigest()
	ref, _, err := ensure(t, d, ensureDigest, PullMissing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("expected no pulls on digest match, got %v", d.pulledRefs)
	}
	if ref != ensureRunRef {
		t.Errorf("expected run ref %s, got %s", ensureRunRef, ref)
	}
}

func TestEnsureImage_MissingPresentMismatchPullsDigestNoRetag(t *testing.T) {
	// The #240 reproduction with the tag already cached: image:tag points at
	// a newer image than the commit was made with. Pull the committed digest
	// and run it; the local tag must stay untouched.
	d := presentMismatch()
	d.pullMakesPresent = true
	ref, _, err := ensure(t, d, ensureDigest, PullMissing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest}) {
		t.Errorf("expected a single pull by digest, got %v", d.pulledRefs)
	}
	if ref != ensureRunRef {
		t.Errorf("expected run ref %s, got %s", ensureRunRef, ref)
	}
}

func TestEnsureImage_MissingAbsentStoredDigestPullsDigestNotTag(t *testing.T) {
	// The #240 reproduction: nothing cached, the tag has moved on the registry.
	d := absentAll()
	d.pullMakesPresent = true
	ref, _, err := ensure(t, d, ensureDigest, PullMissing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest}) {
		t.Errorf("expected a single pull by digest, got %v", d.pulledRefs)
	}
	if ref != ensureRunRef {
		t.Errorf("expected run ref %s, got %s", ensureRunRef, ref)
	}
}

func TestEnsureImage_MissingDigestPullFailsWithLocalWarnsAndUsesTag(t *testing.T) {
	d := presentMismatch()
	d.pullErrFor = map[string]error{ensureDigest: errors.New("manifest unknown")}
	ref, out, err := ensure(t, d, ensureDigest, PullMissing)
	if err != nil {
		t.Fatalf("a present image must survive a failed digest pull, got %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest}) {
		t.Errorf("expected only the digest pull, got %v", d.pulledRefs)
	}
	if !strings.Contains(out, "Warning: unable to pull") || !strings.Contains(out, "differs from the digest recorded") {
		t.Errorf("expected pull warning and digest mismatch warning, got %q", out)
	}
	if ref != ensureRef {
		t.Errorf("expected fallback to run ref %s, got %s", ensureRef, ref)
	}
}

func TestEnsureImage_MissingDigestPullFailsAbsentFallsBackToTag(t *testing.T) {
	d := absentAll()
	d.pullErrFor = map[string]error{ensureDigest: errors.New("manifest unknown")}
	d.pullMakesPresent = true
	ref, _, err := ensure(t, d, ensureDigest, PullMissing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest, ensureRef}) {
		t.Errorf("expected digest pull then tag pull, got %v", d.pulledRefs)
	}
	if ref != ensureRef {
		t.Errorf("expected run ref %s, got %s", ensureRef, ref)
	}
}

func TestEnsureImage_MissingAllPullsFailAbsentErrors(t *testing.T) {
	d := absentAll()
	d.pullErr = errors.New("registry down")
	_, _, err := ensure(t, d, ensureDigest, PullMissing)
	if err == nil {
		t.Fatal("expected error when image is absent and every pull fails")
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest, ensureRef}) {
		t.Errorf("expected digest pull then tag pull, got %v", d.pulledRefs)
	}
}

func TestEnsureImage_MissingUnknownDigestUsesLocal(t *testing.T) {
	// Commits without a digest ("" or the legacy "<nil>") keep #247 behavior.
	for _, stored := range []string{"", "<nil>"} {
		d := &fakeCloneDocker{repoDigests: `["` + ensureOther + `"]`}
		ref, _, err := ensure(t, d, stored, PullMissing)
		if err != nil {
			t.Fatalf("stored=%q unexpected error: %v", stored, err)
		}
		if len(d.pulledRefs) != 0 {
			t.Errorf("stored=%q: expected no pulls, got %v", stored, d.pulledRefs)
		}
		if ref != ensureRef {
			t.Errorf("stored=%q: expected run ref %s, got %s", stored, ensureRef, ref)
		}
	}
}

// ---- never ----

func TestEnsureImage_NeverPresentMatchNoWarning(t *testing.T) {
	d := presentWithDigest()
	ref, out, err := ensure(t, d, ensureDigest, PullNever)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "differs from the digest") {
		t.Errorf("no warning expected on digest match, got %q", out)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("expected no pulls, got %v", d.pulledRefs)
	}
	if ref != ensureRef {
		t.Errorf("expected run ref %s, got %s", ensureRef, ref)
	}
}

func TestEnsureImage_NeverPresentMismatchWarns(t *testing.T) {
	d := presentMismatch()
	ref, out, err := ensure(t, d, ensureDigest, PullNever)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "differs from the digest recorded") {
		t.Errorf("expected mismatch warning, got %q", out)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("--pull never must not pull, got %v", d.pulledRefs)
	}
	if ref != ensureRef {
		t.Errorf("--pull never keeps the local tag image %s, got %s", ensureRef, ref)
	}
}

func TestEnsureImage_NeverTagAbsentDigestPresentRunsDigest(t *testing.T) {
	// The daemon holds the committed digest under another tag; use it.
	d := &fakeCloneDocker{absent: map[string]bool{ensureRef: true}}
	ref, _, err := ensure(t, d, ensureDigest, PullNever)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("--pull never must not pull, got %v", d.pulledRefs)
	}
	if ref != ensureRunRef {
		t.Errorf("expected run ref %s, got %s", ensureRunRef, ref)
	}
}

func TestEnsureImage_NeverPresentUnknownDigestNoWarning(t *testing.T) {
	for _, stored := range []string{"", "<nil>"} {
		d := &fakeCloneDocker{repoDigests: `["` + ensureOther + `"]`}
		_, out, err := ensure(t, d, stored, PullNever)
		if err != nil {
			t.Fatalf("stored=%q unexpected error: %v", stored, err)
		}
		if strings.Contains(out, "differs from the digest") {
			t.Errorf("stored=%q: no warning expected when the commit has no digest, got %q", stored, out)
		}
	}
}

func TestEnsureImage_NeverAbsentErrors(t *testing.T) {
	d := absentAll()
	_, _, err := ensure(t, d, ensureDigest, PullNever)
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
	d := presentWithDigest()
	ref, _, err := ensure(t, d, ensureDigest, PullAlways)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.pulledRefs) != 0 {
		t.Errorf("expected no pulls on digest match, got %v", d.pulledRefs)
	}
	if ref != ensureRunRef {
		t.Errorf("expected run ref %s, got %s", ensureRunRef, ref)
	}
}

func TestEnsureImage_AlwaysMismatchPullsByDigestNoRetag(t *testing.T) {
	d := presentMismatch()
	d.pullMakesPresent = true
	ref, _, err := ensure(t, d, ensureDigest, PullAlways)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest}) {
		t.Errorf("expected a single pull by digest, got %v", d.pulledRefs)
	}
	if ref != ensureRunRef {
		t.Errorf("expected run ref %s, got %s", ensureRunRef, ref)
	}
}

func TestEnsureImage_AlwaysNoStoredDigestPullsTag(t *testing.T) {
	for _, stored := range []string{"", "<nil>"} {
		d := &fakeCloneDocker{repoDigests: `["` + ensureOther + `"]`}
		ref, _, err := ensure(t, d, stored, PullAlways)
		if err != nil {
			t.Fatalf("stored=%q unexpected error: %v", stored, err)
		}
		if !reflect.DeepEqual(d.pulledRefs, []string{ensureRef}) {
			t.Errorf("stored=%q: expected a single pull by tag, got %v", stored, d.pulledRefs)
		}
		if ref != ensureRef {
			t.Errorf("stored=%q: expected run ref %s, got %s", stored, ensureRef, ref)
		}
	}
}

func TestEnsureImage_AlwaysLocalUntaggedPullsByDigest(t *testing.T) {
	// A locally built image has no RepoDigests; the commit does. Pull by digest.
	d := &fakeCloneDocker{repoDigests: "[]", absent: map[string]bool{ensureDigest: true}, pullMakesPresent: true}
	ref, _, err := ensure(t, d, ensureDigest, PullAlways)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest}) {
		t.Errorf("expected a single pull by digest, got %v", d.pulledRefs)
	}
	if ref != ensureRunRef {
		t.Errorf("expected run ref %s, got %s", ensureRunRef, ref)
	}
}

func TestEnsureImage_AlwaysDigestPullFailsWithLocalWarnsAndUsesTag(t *testing.T) {
	d := presentMismatch()
	d.pullErrFor = map[string]error{ensureDigest: errors.New("manifest unknown")}
	ref, out, err := ensure(t, d, ensureDigest, PullAlways)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest}) {
		t.Errorf("expected only the digest pull, got %v", d.pulledRefs)
	}
	if !strings.Contains(out, "Warning: unable to pull") {
		t.Errorf("expected pull warning, got %q", out)
	}
	if ref != ensureRef {
		t.Errorf("expected fallback to run ref %s, got %s", ensureRef, ref)
	}
}

func TestEnsureImage_AlwaysPullFailsWithLocalWarns(t *testing.T) {
	d := &fakeCloneDocker{repoDigests: "[]", pullErr: errors.New("manifest unknown")}
	_, out, err := ensure(t, d, "", PullAlways)
	if err != nil {
		t.Fatalf("a present image must survive a failed refresh, got %v", err)
	}
	if !strings.Contains(out, "Warning: unable to pull") {
		t.Errorf("expected pull warning, got %q", out)
	}
}

func TestEnsureImage_AlwaysPullFailsNoLocalErrors(t *testing.T) {
	d := absentAll()
	d.pullErr = errors.New("registry down")
	_, _, err := ensure(t, d, ensureDigest, PullAlways)
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

func TestEnsureImage_AlwaysAbsentPullsDigestAndSucceeds(t *testing.T) {
	d := absentAll()
	d.pullMakesPresent = true
	ref, _, err := ensure(t, d, ensureDigest, PullAlways)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(d.pulledRefs, []string{ensureDigest}) {
		t.Errorf("expected a single pull by digest, got %v", d.pulledRefs)
	}
	if ref != ensureRunRef {
		t.Errorf("expected run ref %s, got %s", ensureRunRef, ref)
	}
}

func TestRunRef(t *testing.T) {
	cases := []struct{ imageRef, digest, want string }{
		{"repo:tag", "repo@sha256:aaaa", "repo:tag@sha256:aaaa"},
		{"repo:tag", "", "repo:tag"},
		{"repo:tag", "mirror/repo@sha256:aaaa", "mirror/repo:tag@sha256:aaaa"},
		{"repo:tag", "garbage", "repo:tag"},
		{"repo:tag", "@sha256:aaaa", "repo:tag"},
	}
	for _, tc := range cases {
		if got := runRef(tc.imageRef, tc.digest); got != tc.want {
			t.Errorf("runRef(%q, %q) = %q, want %q", tc.imageRef, tc.digest, got, tc.want)
		}
	}
}
