package version

import (
	_ "embed"
	"runtime"
	"strings"
)

//go:embed VERSION
var versionFile string

// Build-time variables set via ldflags
var (
	gitCommit = "unknown"
	gitDate   = "unknown"
)

// Version returns the current version of pgschema
func Version() string {
	return strings.TrimSpace(versionFile)
}

// GitCommit returns the git commit hash
func GitCommit() string {
	return gitCommit
}

// GitDate returns the git commit date
func GitDate() string {
	return gitDate
}

// Platform returns the OS/architecture combination
func Platform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}