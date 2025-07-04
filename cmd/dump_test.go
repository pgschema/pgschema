package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestDumpCommand(t *testing.T) {
	// Test that the command is properly configured
	if DumpCmd.Use != "dump" {
		t.Errorf("Expected Use to be 'dump', got '%s'", DumpCmd.Use)
	}

	if DumpCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	if DumpCmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Test that required flags are defined
	flags := DumpCmd.Flags()
	dbFlag := flags.Lookup("db")
	if dbFlag == nil {
		t.Error("Expected --db flag to be defined")
	}
	userFlag := flags.Lookup("user")
	if userFlag == nil {
		t.Error("Expected --user flag to be defined")
	}

	// Test command validation - should fail without required flags
	cmd := &cobra.Command{}
	cmd.AddCommand(DumpCmd)

	// Reset the flag variables for clean test
	host = "localhost"
	port = 5432
	db = ""
	user = ""

	// Initialize logger for test
	setupLogger()

	err := DumpCmd.RunE(DumpCmd, []string{})
	if err == nil {
		t.Error("Expected command to fail without database connection, but it didn't")
	}
}

func TestDumpCommand_ErrorHandling(t *testing.T) {
	// Store original values
	originalHost := host
	originalPort := port
	originalDb := db
	originalUser := user

	defer func() {
		host = originalHost
		port = originalPort
		db = originalDb
		user = originalUser
	}()

	// Test with invalid connection parameters
	host = "localhost"
	port = 9999
	db = "nonexistent"
	user = "invalid"

	err := runDump(nil, nil)
	if err == nil {
		t.Error("Expected error with unreachable database, but got nil")
	}
}
