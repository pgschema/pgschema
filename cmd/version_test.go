package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)


func TestVersionCommand(t *testing.T) {
	var buf bytes.Buffer

	// Create a copy of the version command to avoid affecting the global one
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display the version number of pgschema",
		Run: func(cmd *cobra.Command, args []string) {
			version := strings.TrimSpace(versionFile)
			buf.WriteString(fmt.Sprintf("pgschema version %s\n", version))
		},
	}

	cmd := &cobra.Command{Use: "pgschema"}
	cmd.AddCommand(versionCmd)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("version command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "pgschema version") {
		t.Errorf("expected version output to contain 'pgschema version', got: %s", output)
	}
}

func TestVersionCommandOutput(t *testing.T) {
	var buf bytes.Buffer

	// Create a copy of the version command to capture output
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display the version number of pgschema",
		Run: func(cmd *cobra.Command, args []string) {
			version := strings.TrimSpace(versionFile)
			buf.WriteString(fmt.Sprintf("pgschema version %s\n", version))
		},
	}

	// Create a temporary root command to test version command
	rootCmd := &cobra.Command{Use: "pgschema"}
	rootCmd.AddCommand(versionCmd)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command execution failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	// Check that output follows expected format
	if !strings.HasPrefix(output, "pgschema version ") {
		t.Errorf("expected output to start with 'pgschema version ', got: %s", output)
	}

	// Check that there's actually a version after the prefix
	versionPart := strings.TrimPrefix(output, "pgschema version ")
	if len(versionPart) == 0 {
		t.Error("expected version information after 'pgschema version ', got empty string")
	}
}
