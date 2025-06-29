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

	// Test that required flags are defined
	flags := InspectCmd.Flags()
	dbnameFlag := flags.Lookup("dbname")
	if dbnameFlag == nil {
		t.Error("Expected --dbname flag to be defined")
	}
	usernameFlag := flags.Lookup("username")
	if usernameFlag == nil {
		t.Error("Expected --username flag to be defined")
	}

	// Test command validation - should fail without required flags
	cmd := &cobra.Command{}
	cmd.AddCommand(InspectCmd)

	// Reset the flag variables for clean test
	host = "localhost"
	port = 5432
	dbname = ""
	username = ""

	// Initialize logger for test
	setupLogger()

	err := InspectCmd.RunE(InspectCmd, []string{})
	if err == nil {
		t.Error("Expected command to fail without database connection, but it didn't")
	}
}

func TestInspectCommand_ErrorHandling(t *testing.T) {
	// Store original values
	originalHost := host
	originalPort := port
	originalDbname := dbname
	originalUsername := username
	
	defer func() {
		host = originalHost
		port = originalPort
		dbname = originalDbname
		username = originalUsername
	}()

	// Test with invalid connection parameters
	host = "localhost"
	port = 9999
	dbname = "nonexistent"
	username = "invalid"

	err := runInspect(nil, nil)
	if err == nil {
		t.Error("Expected error with unreachable database, but got nil")
	}
}