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
	return PreRunEWithEnvVarsAndConnection(dbPtr, userPtr, nil, nil)
}

// PreRunEWithEnvVarsAndConnection creates a PreRunE function that validates database connection parameters
// It checks environment variables if the corresponding flags weren't explicitly set
// This version also handles optional host, port, and application name parameters
func PreRunEWithEnvVarsAndConnection(dbPtr, userPtr *string, hostPtr *string, portPtr *int) func(*cobra.Command, []string) error {
	return PreRunEWithEnvVarsAndConnectionAndApp(dbPtr, userPtr, hostPtr, portPtr, nil)
}

// PreRunEWithEnvVarsAndConnectionAndApp creates a PreRunE function that validates database connection parameters
// It checks environment variables if the corresponding flags weren't explicitly set
// This version handles all optional connection parameters including application name
func PreRunEWithEnvVarsAndConnectionAndApp(dbPtr, userPtr *string, hostPtr *string, portPtr *int, appNamePtr *string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Check if required values are available from environment variables
		if GetEnvWithDefault("PGDATABASE", "") != "" && !cmd.Flags().Changed("db") {
			*dbPtr = GetEnvWithDefault("PGDATABASE", "")
		}
		if GetEnvWithDefault("PGUSER", "") != "" && !cmd.Flags().Changed("user") {
			*userPtr = GetEnvWithDefault("PGUSER", "")
		}

		// Check optional host and port if pointers provided
		if hostPtr != nil && GetEnvWithDefault("PGHOST", "") != "" && !cmd.Flags().Changed("host") {
			*hostPtr = GetEnvWithDefault("PGHOST", "")
		}
		if portPtr != nil && GetEnvIntWithDefault("PGPORT", 0) != 0 && !cmd.Flags().Changed("port") {
			*portPtr = GetEnvIntWithDefault("PGPORT", 0)
		}

		// Check optional application name if pointer provided
		if appNamePtr != nil && GetEnvWithDefault("PGAPPNAME", "") != "" && !cmd.Flags().Changed("application-name") {
			*appNamePtr = GetEnvWithDefault("PGAPPNAME", "")
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