package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// App returns the pgschema application version
func App() string {
	return strings.TrimSpace(versionFile)
}

// PlanFormat returns the plan format version
func PlanFormat() string {
	return "1.0.0"
}