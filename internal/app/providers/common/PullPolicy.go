// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package common

import (
	"errors"
	"fmt"
	"strings"
)

// PullPolicy controls how `dit run` and `dit clone` resolve a container
// image on the docker daemon the CLI talks to. It mirrors docker's own
// --pull flag.
type PullPolicy string

const (
	inspectNull   = "null"  // docker inspect output for an absent field
	unknownDigest = "<nil>" // how older metadata loaded a missing digest
)

const (
	// PullMissing uses the image already on the daemon and pulls only when
	// it is absent. When the commit records a digest (dit clone), "the
	// image" is that digest, not whatever image:tag points at today. This
	// is the default.
	PullMissing PullPolicy = "missing"
	// PullAlways refreshes the image from the registry. With a recorded
	// digest it behaves like PullMissing (the digest is already exact);
	// without one it re-pulls image:tag.
	PullAlways PullPolicy = "always"
	// PullNever uses local images only and fails when nothing usable is
	// present.
	PullNever PullPolicy = "never"
)

// ParsePullPolicy validates a --pull flag value. The empty string means the
// default policy.
func ParsePullPolicy(s string) (PullPolicy, error) {
	switch PullPolicy(s) {
	case "":
		return PullMissing, nil
	case PullMissing, PullAlways, PullNever:
		return PullPolicy(s), nil
	}
	return "", fmt.Errorf("invalid --pull value %q (want missing, always or never)", s)
}

// ImagePuller is the docker surface EnsureImage needs. Each provider
// package's dockerClient interface satisfies it structurally.
type ImagePuller interface {
	InspectImage(image string) (string, error)
	GetValFromImage(image string, key ...string) string
	Pull(image string) (string, error)
}

// RepoDigests parses the raw `[0].RepoDigests` text from `docker inspect`
// (for example `["redis@sha256:abc", "mirror/redis@sha256:abc"]`) into a
// clean slice. An image built locally and never pushed has none.
func RepoDigests(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == inspectNull {
		return nil
	}
	raw = strings.Trim(raw, "[]")
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.Trim(strings.TrimSpace(part), `"`)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// runRef turns image:tag plus a RepoDigests entry ("repo@sha256:…") into
// "repo:tag@sha256:…". Docker resolves such a reference by the digest alone,
// so the container starts from exactly the committed image while the tag
// still travels along for the repository metadata. The repository name is
// taken from the digest entry (it is the name the digest is known under on
// this daemon, normally the same as the image's). It returns imageRef
// unchanged when digest is empty or not of the repo@algo:hex form.
func runRef(imageRef string, digest string) string {
	at := strings.LastIndex(digest, "@")
	if at < 0 {
		return imageRef
	}
	repo, d := digest[:at], digest[at+1:]
	if repo == "" || !strings.Contains(d, ":") {
		return imageRef
	}
	ref := repo
	if colon := strings.LastIndex(imageRef, ":"); colon >= 0 {
		ref += imageRef[colon:]
	}
	return ref + "@" + d
}

// EnsureImage makes the image usable on the local docker daemon according
// to policy and returns the reference the container must be started from.
// imageRef is image:tag. storedDigest is the RepoDigests entry recorded in
// the commit being cloned, or "" when unknown (dit run, or a commit written
// without one). The returned reference is image:tag@digest when the
// committed digest is used (issue #240) and image:tag otherwise. It prints
// its own warnings; a non-nil error means nothing usable is present and the
// caller should exit.
func EnsureImage(d ImagePuller, imageRef string, storedDigest string, policy PullPolicy) (string, error) {
	stored := storedDigest
	if stored == unknownDigest {
		// Older commit metadata loaded a missing digest as the literal "<nil>".
		stored = ""
	}
	digestRef := runRef(imageRef, stored)
	if digestRef == imageRef {
		stored = "" // nothing usable was recorded
	}
	present := imagePresent(d, imageRef)

	if policy == PullNever {
		if present {
			if stored != "" {
				warnOnMismatch(d, imageRef, stored, "--pull never keeps the local image")
			}
			return imageRef, nil
		}
		if stored != "" && imagePresent(d, stored) {
			return digestRef, nil
		}
		return "", fmt.Errorf("image %s is not present locally and --pull never was given", imageRef)
	}

	// missing / always with a recorded digest: the digest is the image.
	if stored != "" {
		if imagePresent(d, stored) {
			return digestRef, nil
		}
		err := pullRef(d, stored)
		if err == nil {
			return digestRef, nil
		}
		if present {
			fmt.Printf("Warning: unable to pull the image recorded in the commit %s (%v); using local image %s\n", stored, err, imageRef)
			warnOnMismatch(d, imageRef, stored, "the local image is used anyway")
			return imageRef, nil
		}
		if err := pullRef(d, imageRef); err != nil {
			return "", fmt.Errorf("unable to pull image %s or %s: %v", stored, imageRef, err)
		}
		return imageRef, nil
	}

	if policy == PullAlways {
		err := pullRef(d, imageRef)
		if err == nil {
			return imageRef, nil
		}
		if present {
			fmt.Printf("Warning: unable to pull %s (%v); using local image\n", imageRef, err)
			return imageRef, nil
		}
		return "", fmt.Errorf("unable to pull image %s: %v", imageRef, err)
	}

	// PullMissing without a recorded digest
	if present {
		return imageRef, nil
	}
	if err := pullRef(d, imageRef); err != nil {
		return "", fmt.Errorf("image %s not found locally and pull failed: %v", imageRef, err)
	}
	return imageRef, nil
}

func imagePresent(d ImagePuller, imageRef string) bool {
	out, err := d.InspectImage(imageRef)
	return err == nil && strings.TrimSpace(out) != ""
}

// warnOnMismatch prints a warning when the local image:tag has RepoDigests
// and none of them is the digest recorded in the commit.
func warnOnMismatch(d ImagePuller, imageRef string, stored string, consequence string) {
	local := RepoDigests(d.GetValFromImage(imageRef, "RepoDigests"))
	if len(local) == 0 {
		return
	}
	for _, l := range local {
		if l == stored {
			return
		}
	}
	fmt.Printf("Warning: local image %s (%s) differs from the digest recorded in the commit (%s); %s\n",
		imageRef, strings.Join(local, ","), stored, consequence)
}

func pullRef(d ImagePuller, ref string) error {
	if _, err := d.Pull(ref); err != nil {
		return err
	}
	if !imagePresent(d, ref) {
		return errors.New("image still missing after pull")
	}
	return nil
}
