package version

import (
	"fmt"
)

var (
	// Major is the current major version number
	Major = 0
	// Minor is the current minor version number
	Minor = 1
	// Patch is the current patch version number
	Patch = 0
)

func Version() string {
	return fmt.Sprintf("v%d.%d.%d-dev", Major, Minor, Patch)
}
