package local

import (
	"fmt"
	"regexp"
	"strings"
)

// validateRepositoryName validates that a repository name follows Docker volume naming conventions
// Names must contain only lowercase letters, numbers, and hyphens
// Cannot start or end with a hyphen
// Cannot contain underscores (conflicts with internal volume naming: <repo>_v<N>)
func validateRepositoryName(name string) error {
	// Check for slash (already validated but keep for clarity)
	if strings.Contains(name, "/") {
		return fmt.Errorf("repository name '%s' is invalid: cannot contain slashes", name)
	}

	// Check for underscore (conflicts with volume naming scheme <repo>_v<N>)
	if strings.Contains(name, "_") {
		return fmt.Errorf("repository name '%s' is invalid: cannot contain underscores (conflicts with volume naming)", name)
	}

	// Check for uppercase letters
	if name != strings.ToLower(name) {
		return fmt.Errorf("repository name '%s' is invalid: must be lowercase", name)
	}

	// Validate pattern: lowercase letters, numbers, hyphens
	// Must start and end with alphanumeric character
	pattern := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	if !pattern.MatchString(name) {
		return fmt.Errorf("repository name '%s' is invalid: must contain only lowercase letters, numbers, and hyphens (cannot start or end with hyphen)", name)
	}

	return nil
}
