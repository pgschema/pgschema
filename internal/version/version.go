package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version returns the current version of pgschema
func Version() string {
	return strings.TrimSpace(versionFile)
}