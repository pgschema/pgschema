// Package postgres provides PostgreSQL functionality for desired state management.
// This file defines the interface for desired state providers (embedded or external databases).
package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"
)

// DesiredStateProvider is an interface that abstracts the desired state database provider.
// It can be implemented by either embedded PostgreSQL or an external database connection.
type DesiredStateProvider interface {
	// GetConnectionDetails returns connection details for IR inspection
	// Returns: host, port, database, username, password
	GetConnectionDetails() (string, int, string, string, string)

	// GetSchemaName returns the actual schema name to inspect.
	// For embedded postgres: returns the temporary schema name (pgschema_tmp_*)
	// For external database: returns the temporary schema name (pgschema_tmp_*)
	GetSchemaName() string

	// ApplySchema applies the desired state SQL to a schema.
	// For embedded postgres: resets the schema (drop/recreate)
	// For external database: creates temporary schema with timestamp suffix
	ApplySchema(ctx context.Context, schema string, sql string) error

	// Stop performs cleanup.
	// For embedded postgres: stops instance and removes temp directory
	// For external database: drops temporary schema (best effort) and closes connection
	Stop() error
}

// GenerateTempSchemaName creates a unique temporary schema name for plan operations.
// The format is: pgschema_tmp_YYYYMMDD_HHMMSS_RRRRRRRR
// where RRRRRRRR is a random 8-character hex string for uniqueness.
// The "_tmp_" marker makes it distinctive and prevents accidental matching with user schemas.
//
// Example: pgschema_tmp_20251030_154501_a3f9d2e1
//
// Panics if random number generation fails (indicates serious system issue).
func GenerateTempSchemaName() string {
	timestamp := time.Now().Format("20060102_150405")

	// Add random suffix for uniqueness (4 bytes = 8 hex characters)
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		// If crypto/rand fails, something is seriously wrong with the system
		panic(fmt.Sprintf("failed to generate random schema name: %v", err))
	}
	randomSuffix := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("pgschema_tmp_%s_%s", timestamp, randomSuffix)
}

// stripSchemaQualifications removes schema qualifications from SQL statements for the specified target schema.
//
// Purpose:
// When applying user-provided SQL to temporary schemas during the plan command, we need to ensure
// that objects are created in the temporary schema (e.g., pgschema_tmp_20251030_154501_123456789)
// rather than in explicitly qualified schemas. PostgreSQL's search_path only affects unqualified
// object names - explicit schema qualifications always override search_path.
//
// Input SQL Sources:
// - pgschema dump command produces schema-agnostic output (no schema qualifications for target schema)
// - Users may manually edit SQL files and add schema qualifications (e.g., public.table)
// - Users may provide SQL from other sources that contains schema qualifications
//
// Behavior:
// This function strips schema qualifications ONLY for the target schema (specified by schemaName),
// while preserving qualifications for other schemas. This allows:
// 1. Target schema objects to be created in temporary schemas via search_path
// 2. Cross-schema references to be preserved correctly
//
// Examples:
// When target schema is "public":
// - public.employees -> employees (stripped - will use search_path)
// - "public".employees -> employees (stripped - handles quoted identifiers)
// - public."employees" -> "employees" (stripped - preserves quoted object names)
// - other_schema.users -> other_schema.users (preserved - cross-schema reference)
//
// It handles both quoted and unquoted schema names:
// - public.table -> table
// - "public".table -> table
// - public."table" -> "table"
// - "public"."table" -> "table"
//
// Only qualifications matching the specified schemaName are stripped.
// All other schema qualifications are preserved as intentional cross-schema references.
func stripSchemaQualifications(sql string, schemaName string) string {
	if schemaName == "" {
		return sql
	}

	// Escape the schema name for use in regex
	escapedSchema := regexp.QuoteMeta(schemaName)

	// Pattern matches: optional quote + schemaName + optional quote + dot + captured object name
	// This handles all four combinations of quoted/unquoted schema and object names

	// Pattern for unquoted identifier after dot
	pattern1 := fmt.Sprintf(`"?%s"?\.([a-zA-Z_][a-zA-Z0-9_$]*)`, escapedSchema)
	re1 := regexp.MustCompile(pattern1)

	// Pattern for quoted identifier after dot
	pattern2 := fmt.Sprintf(`"?%s"?\.(\"[^"]+\")`, escapedSchema)
	re2 := regexp.MustCompile(pattern2)

	result := sql
	result = re1.ReplaceAllString(result, "$1")
	result = re2.ReplaceAllString(result, "$1")

	return result
}
