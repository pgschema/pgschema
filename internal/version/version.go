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
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// Version returns the current version of pgschema
func Version() string {
	return strings.TrimSpace(versionFile)
}

// GetGitCommit returns the git commit hash
func GetGitCommit() string {
	return GitCommit
}

// GetBuildDate returns the git commit date
func GetBuildDate() string {
	return BuildDate
}

// Platform returns the OS/architecture combination
func Platform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
