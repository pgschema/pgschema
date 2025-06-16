package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestInspectCommand(t *testing.T) {
	// Test that the command is properly configured
	if InspectCmd.Use != "inspect" {
		t.Errorf("Expected Use to be 'inspect', got '%s'", InspectCmd.Use)
	}

	if InspectCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	if InspectCmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Test that the --dsn flag is required
	flags := InspectCmd.Flags()
	dsnFlag := flags.Lookup("dsn")
	if dsnFlag == nil {
		t.Error("Expected --dsn flag to be defined")
	}

	// Test command validation - should fail without --dsn
	cmd := &cobra.Command{}
	cmd.AddCommand(InspectCmd)

	// Reset the dsn variable for clean test
	dsn = ""

	// Initialize logger for test
	setupLogger()

	err := InspectCmd.RunE(InspectCmd, []string{})
	if err == nil {
		t.Error("Expected command to fail without database connection, but it didn't")
	}
}

func TestInspectCommand_ErrorHandling(t *testing.T) {
	// Test with invalid DSN to ensure proper error handling
	originalDSN := dsn
	dsn = "invalid://connection/string"
	defer func() { dsn = originalDSN }()

	err := runInspect(nil, nil)
	if err == nil {
		t.Error("Expected error with invalid DSN, but got nil")
	}

	// Test with DSN that fails to connect
	dsn = "postgres://invalid:invalid@localhost:9999/nonexistent"
	err = runInspect(nil, nil)
	if err == nil {
		t.Error("Expected error with unreachable database, but got nil")
	}
}