package app

import (
	"fmt"
	"strconv"
	"strings"
)

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
	fmt.Printf("DEBUG Version.FromString: version='%s', cleaned='%s', split result=%v, length=%d\n", version, cleanVersion, v, len(v))
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
