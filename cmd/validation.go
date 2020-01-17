package cmd

import (
	"fmt"
	"regexp"
)

// Regex's used to validate inputs
var (
	validConfigPathFormat = regexp.MustCompile(`^[\/]*[\w\.\-]+(\/[\w\.\-]+)*$`)
)

//nolint:lll
func validateConfigPathName(configPath string) error {
	if !validConfigPathFormat.MatchString(configPath) {
		return fmt.Errorf(`failed to validate configuration path name '%s'
Only alphanumeric, dashes, forwardslashes, fullstops and underscores are allowed for configuration path names`, configPath)
	}

	return nil
}
