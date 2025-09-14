package cmd

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestDotenvLoading(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()

	// Change to temp directory
	err := os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Restore original directory after test
	defer func() {
		os.Chdir(originalDir)
	}()

	// Test 1: Load .env file with PGPASSWORD
	t.Run("LoadEnvFile", func(t *testing.T) {
		// Clean environment first
		os.Unsetenv("PGPASSWORD")

		// Create .env file
		envContent := "PGPASSWORD=test_password_123\n"
		err := os.WriteFile(".env", []byte(envContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create .env file: %v", err)
		}

		// Load .env file
		err = godotenv.Load()
		if err != nil {
			t.Fatalf("Failed to load .env file: %v", err)
		}

		// Verify PGPASSWORD is set
		password := os.Getenv("PGPASSWORD")
		if password != "test_password_123" {
			t.Errorf("Expected PGPASSWORD='test_password_123', got '%s'", password)
		}

		// Cleanup
		os.Remove(".env")
		os.Unsetenv("PGPASSWORD")
	})

	// Test 2: Missing .env file should not cause errors
	t.Run("MissingEnvFile", func(t *testing.T) {
		// Clean environment first
		os.Unsetenv("PGPASSWORD")

		// Ensure no .env file exists
		os.Remove(".env")

		// Load .env file (should not error)
		err := godotenv.Load()
		if err == nil {
			t.Error("Expected error when loading non-existent .env file, but got nil")
		}

		// PGPASSWORD should be empty
		password := os.Getenv("PGPASSWORD")
		if password != "" {
			t.Errorf("Expected PGPASSWORD to be empty, got '%s'", password)
		}
	})

	// Test 3: Environment variable priority
	t.Run("EnvVarPriority", func(t *testing.T) {
		// Set PGPASSWORD in environment first
		os.Setenv("PGPASSWORD", "env_password")

		// Create .env file with different password
		envContent := "PGPASSWORD=dotenv_password\n"
		err := os.WriteFile(".env", []byte(envContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create .env file: %v", err)
		}

		// Load .env file - should NOT override existing env var
		err = godotenv.Load()
		if err != nil {
			t.Fatalf("Failed to load .env file: %v", err)
		}

		// Should still have the original environment value
		password := os.Getenv("PGPASSWORD")
		if password != "env_password" {
			t.Errorf("Expected PGPASSWORD='env_password' (existing env var should take precedence), got '%s'", password)
		}

		// Cleanup
		os.Remove(".env")
		os.Unsetenv("PGPASSWORD")
	})

	// Test 4: .env overrides when no existing env var
	t.Run("DotenvOverridesWhenNoEnvVar", func(t *testing.T) {
		// Clean environment first
		os.Unsetenv("PGPASSWORD")

		// Create .env file
		envContent := "PGPASSWORD=dotenv_only_password\n"
		err := os.WriteFile(".env", []byte(envContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create .env file: %v", err)
		}

		// Load .env file
		err = godotenv.Load()
		if err != nil {
			t.Fatalf("Failed to load .env file: %v", err)
		}

		// Should have the dotenv value
		password := os.Getenv("PGPASSWORD")
		if password != "dotenv_only_password" {
			t.Errorf("Expected PGPASSWORD='dotenv_only_password', got '%s'", password)
		}

		// Cleanup
		os.Remove(".env")
		os.Unsetenv("PGPASSWORD")
	})

	// Test 5: All PostgreSQL environment variables
	t.Run("AllPostgreSQLEnvVars", func(t *testing.T) {
		// Clean environment first
		envVars := []string{"PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD", "PGAPPNAME"}
		for _, envVar := range envVars {
			os.Unsetenv(envVar)
		}

		// Create .env file with all variables
		envContent := `PGHOST=test.example.com
PGPORT=5433
PGDATABASE=testdb
PGUSER=testuser
PGPASSWORD=testpass
PGAPPNAME=test-pgschema
`
		err := os.WriteFile(".env", []byte(envContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create .env file: %v", err)
		}

		// Load .env file
		err = godotenv.Load()
		if err != nil {
			t.Fatalf("Failed to load .env file: %v", err)
		}

		// Verify all variables are loaded
		expectedValues := map[string]string{
			"PGHOST":     "test.example.com",
			"PGPORT":     "5433",
			"PGDATABASE": "testdb",
			"PGUSER":     "testuser",
			"PGPASSWORD": "testpass",
			"PGAPPNAME":  "test-pgschema",
		}

		for envVar, expected := range expectedValues {
			actual := os.Getenv(envVar)
			if actual != expected {
				t.Errorf("Expected %s='%s', got '%s'", envVar, expected, actual)
			}
		}

		// Cleanup
		os.Remove(".env")
		for _, envVar := range envVars {
			os.Unsetenv(envVar)
		}
	})
}
