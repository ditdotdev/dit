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
	// PullMissing uses the local image when present and pulls only when it
	// is absent. This is the default.
	PullMissing PullPolicy = "missing"
	// PullAlways refreshes the image from the registry unless the local copy
	// already matches the digest recorded in the commit being cloned.
	PullAlways PullPolicy = "always"
	// PullNever uses the local image only and fails when it is absent.
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
	Tag(source string, target string) (string, error)
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

// EnsureImage makes imageRef (image:tag) usable on the local docker daemon
// according to policy. storedDigest is the RepoDigests entry recorded in the
// commit metadata, or "" when unknown (dit run, or a commit written without
// one). It prints its own warnings; a non-nil error means the image is not
// usable and the caller should exit.
func EnsureImage(d ImagePuller, imageRef string, storedDigest string, policy PullPolicy) error {
	stored := storedDigest
	if stored == unknownDigest {
		// Older commit metadata loaded a missing digest as the literal "<nil>".
		stored = ""
	}
	present := imagePresent(d, imageRef)

	switch policy {
	case PullNever:
		if !present {
			return fmt.Errorf("image %s is not present locally and --pull never was given", imageRef)
		}
		if stored != "" {
			if match, local := localDigestMatches(d, imageRef, stored); !match && len(local) > 0 {
				fmt.Printf("Warning: local image %s (%s) differs from the digest recorded in the commit (%s); --pull never keeps the local image\n",
					imageRef, strings.Join(local, ","), stored)
			}
		}
		return nil

	case PullAlways:
		if present && stored != "" {
			if match, _ := localDigestMatches(d, imageRef, stored); match {
				return nil
			}
		}
		err := refreshImage(d, imageRef, stored)
		if err == nil {
			return nil
		}
		if present {
			fmt.Printf("Warning: unable to pull %s (%v); using local image\n", imageRef, err)
			return nil
		}
		return fmt.Errorf("unable to pull image %s: %v", imageRef, err)

	default: // PullMissing
		if present {
			return nil
		}
		if err := pullRef(d, imageRef); err != nil {
			return fmt.Errorf("image %s not found locally and pull failed: %v", imageRef, err)
		}
		return nil
	}
}

func imagePresent(d ImagePuller, imageRef string) bool {
	out, err := d.InspectImage(imageRef)
	return err == nil && strings.TrimSpace(out) != ""
}

// localDigestMatches reports whether any RepoDigests entry of the local image
// equals stored, and returns the local entries for diagnostics.
func localDigestMatches(d ImagePuller, imageRef string, stored string) (bool, []string) {
	local := RepoDigests(d.GetValFromImage(imageRef, "RepoDigests"))
	for _, l := range local {
		if l == stored {
			return true, local
		}
	}
	return false, local
}

// refreshImage pulls the commit's digest and points imageRef at it, so the
// container started from image:tag is exactly the committed image. Pulling
// by digest does not move the tag by itself, hence the docker tag. Without a
// stored digest, or when the digest pull fails, it pulls the tag instead.
func refreshImage(d ImagePuller, imageRef string, stored string) error {
	if stored != "" {
		if _, err := d.Pull(stored); err == nil {
			if _, err := d.Tag(stored, imageRef); err == nil {
				return nil
			}
		}
	}
	return pullRef(d, imageRef)
}

func pullRef(d ImagePuller, imageRef string) error {
	if _, err := d.Pull(imageRef); err != nil {
		return err
	}
	if !imagePresent(d, imageRef) {
		return errors.New("image still missing after pull")
	}
	return nil
}
