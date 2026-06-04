package app

import (
	"strconv"
	"strings"
)

// DitVersion will be set at build time via -ldflags
// This is the version used for both the CLI and the server Docker image
var DitVersion = "dev"

type Version struct {
	major int
	minor int
	micro int
}

func (Version) FromString(version string) Version {
	// Strip 'v' prefix if present
	cleanVersion := version
	if strings.HasPrefix(version, "v") {
		cleanVersion = strings.TrimPrefix(version, "v")
	}

	v := strings.Split(cleanVersion, ".")
	if len(v) < 3 {
		// Handle malformed version strings gracefully
		major, _ := strconv.Atoi(v[0])
		minor := 0
		micro := 0
		if len(v) > 1 {
			minor, _ = strconv.Atoi(v[1])
		}
		if len(v) > 2 {
			micro, _ = strconv.Atoi(v[2])
		}
		return Version{major, minor, micro}
	}
	major, _ := strconv.Atoi(v[0])
	minor, _ := strconv.Atoi(v[1])
	micro, _ := strconv.Atoi(v[2])
	return Version{major, minor, micro}
}

func (from Version) Compare(to Version) int {
	if from.major > to.major {
		return 1
	}
	if from.major < to.major {
		return -1
	}
	if from.minor > to.minor {
		return 1
	}
	if from.minor < to.minor {
		return -1
	}
	if from.micro > to.micro {
		return 1
	}
	if from.micro < to.micro {
		return -1
	}
	return 0
}
