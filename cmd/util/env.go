package util

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

// GetEnvWithDefault returns the value of an environment variable or a default value if not set
func GetEnvWithDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvIntWithDefault returns the value of an environment variable as int or a default value if not set
func GetEnvIntWithDefault(envVar string, defaultValue int) int {
	if value := os.Getenv(envVar); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// PreRunEWithEnvVars creates a PreRunE function that validates required database connection parameters
// It checks environment variables if the corresponding flags weren't explicitly set
func PreRunEWithEnvVars(dbPtr, userPtr *string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Check if required values are available from environment variables
		if GetEnvWithDefault("PGDATABASE", "") != "" && !cmd.Flags().Changed("db") {
			*dbPtr = GetEnvWithDefault("PGDATABASE", "")
		}
		if GetEnvWithDefault("PGUSER", "") != "" && !cmd.Flags().Changed("user") {
			*userPtr = GetEnvWithDefault("PGUSER", "")
		}

		// Now validate that we have the required values
		if *dbPtr == "" {
			return fmt.Errorf("database name is required (use --db flag or PGDATABASE environment variable)")
		}
		if *userPtr == "" {
			return fmt.Errorf("database user is required (use --user flag or PGUSER environment variable)")
		}

		return nil
	}
}