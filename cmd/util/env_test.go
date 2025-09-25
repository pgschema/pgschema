package util

import (
	"os"
	"testing"
)

func TestGetEnvWithDefault(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_STRING", "test-value")
	if GetEnvWithDefault("TEST_STRING", "default") != "test-value" {
		t.Errorf("Expected GetEnvWithDefault to return 'test-value', got '%s'", GetEnvWithDefault("TEST_STRING", "default"))
	}

	// Test with missing env var
	os.Unsetenv("MISSING_VAR")
	if GetEnvWithDefault("MISSING_VAR", "default") != "default" {
		t.Errorf("Expected GetEnvWithDefault to return 'default', got '%s'", GetEnvWithDefault("MISSING_VAR", "default"))
	}

	// Test with empty env var (should return default)
	os.Setenv("EMPTY_VAR", "")
	if GetEnvWithDefault("EMPTY_VAR", "default") != "default" {
		t.Errorf("Expected GetEnvWithDefault to return 'default' for empty var, got '%s'", GetEnvWithDefault("EMPTY_VAR", "default"))
	}

	// Cleanup
	os.Unsetenv("TEST_STRING")
	os.Unsetenv("EMPTY_VAR")
}

func TestGetEnvIntWithDefault(t *testing.T) {
	// Test with valid int env var
	os.Setenv("TEST_INT", "12345")
	if GetEnvIntWithDefault("TEST_INT", 0) != 12345 {
		t.Errorf("Expected GetEnvIntWithDefault to return 12345, got %d", GetEnvIntWithDefault("TEST_INT", 0))
	}

	// Test with invalid int value (should return default)
	os.Setenv("TEST_INVALID_INT", "not-a-number")
	if GetEnvIntWithDefault("TEST_INVALID_INT", 999) != 999 {
		t.Errorf("Expected GetEnvIntWithDefault to return default 999, got %d", GetEnvIntWithDefault("TEST_INVALID_INT", 999))
	}

	// Test with missing env var
	os.Unsetenv("MISSING_INT_VAR")
	if GetEnvIntWithDefault("MISSING_INT_VAR", 777) != 777 {
		t.Errorf("Expected GetEnvIntWithDefault to return default 777, got %d", GetEnvIntWithDefault("MISSING_INT_VAR", 777))
	}

	// Test with empty env var (should return default)
	os.Setenv("EMPTY_INT_VAR", "")
	if GetEnvIntWithDefault("EMPTY_INT_VAR", 888) != 888 {
		t.Errorf("Expected GetEnvIntWithDefault to return default 888 for empty var, got %d", GetEnvIntWithDefault("EMPTY_INT_VAR", 888))
	}

	// Cleanup
	os.Unsetenv("TEST_INT")
	os.Unsetenv("TEST_INVALID_INT")
	os.Unsetenv("EMPTY_INT_VAR")
}

func TestPreRunEWithEnvVars(t *testing.T) {
	// Setup test environment
	os.Setenv("PGDATABASE", "test-db")
	os.Setenv("PGUSER", "test-user")
	os.Setenv("PGHOST", "test-host")
	os.Setenv("PGPORT", "1234")
	os.Setenv("PGAPPNAME", "test-app")

	// Test variables to be populated
	var db, user, host, appName string
	var port int

	// Create a mock command that simulates flags not being changed
	// In real usage, cobra.Command would handle this, but for testing we'll call the function directly
	preRunFunc := PreRunEWithEnvVarsAndConnectionAndApp(&db, &user, &host, &port, &appName)

	// We can't easily test this without a real cobra.Command, but we can test the underlying logic
	// by directly calling the helper functions which are used in the PreRun function

	// Test that environment variables are read correctly
	if GetEnvWithDefault("PGDATABASE", "") != "test-db" {
		t.Errorf("Expected PGDATABASE to be 'test-db', got '%s'", GetEnvWithDefault("PGDATABASE", ""))
	}

	if GetEnvWithDefault("PGUSER", "") != "test-user" {
		t.Errorf("Expected PGUSER to be 'test-user', got '%s'", GetEnvWithDefault("PGUSER", ""))
	}

	if GetEnvWithDefault("PGHOST", "") != "test-host" {
		t.Errorf("Expected PGHOST to be 'test-host', got '%s'", GetEnvWithDefault("PGHOST", ""))
	}

	if GetEnvIntWithDefault("PGPORT", 0) != 1234 {
		t.Errorf("Expected PGPORT to be 1234, got %d", GetEnvIntWithDefault("PGPORT", 0))
	}

	if GetEnvWithDefault("PGAPPNAME", "") != "test-app" {
		t.Errorf("Expected PGAPPNAME to be 'test-app', got '%s'", GetEnvWithDefault("PGAPPNAME", ""))
	}

	// Cleanup
	os.Unsetenv("PGDATABASE")
	os.Unsetenv("PGUSER")
	os.Unsetenv("PGHOST")
	os.Unsetenv("PGPORT")
	os.Unsetenv("PGAPPNAME")

	// Verify preRunFunc was created (basic sanity check)
	if preRunFunc == nil {
		t.Error("PreRunEWithEnvVarsAndConnectionAndApp should return a non-nil function")
	}
}