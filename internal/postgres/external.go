// Package postgres provides external PostgreSQL database functionality for desired state management.
package postgres

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// ExternalDatabase manages an external PostgreSQL database for desired state validation.
// It creates temporary schemas with timestamp suffixes to avoid conflicts.
type ExternalDatabase struct {
	db                 *sql.DB
	host               string
	port               int
	database           string
	username           string
	password           string
	tempSchema         string // Temporary schema name with timestamp suffix
	targetMajorVersion int    // Expected major version (from target database)
}

// ExternalDatabaseConfig holds configuration for connecting to an external database
type ExternalDatabaseConfig struct {
	Host               string
	Port               int
	Database           string
	Username           string
	Password           string
	TargetMajorVersion int // Expected major version to match
}

// NewExternalDatabase creates a new external database connection for desired state validation.
// It validates the connection, checks version compatibility, and generates a temporary schema name.
func NewExternalDatabase(config *ExternalDatabaseConfig) (*ExternalDatabase, error) {
	// Build connection string
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=prefer",
		config.Username, config.Password, config.Host, config.Port, config.Database)

	// Connect to database
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to external database: %w", err)
	}

	// Test the connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping external database: %w", err)
	}

	// Detect version and validate compatibility
	majorVersion, err := detectMajorVersion(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to detect PostgreSQL version: %w", err)
	}

	// Validate version compatibility (require exact major version match)
	if majorVersion != config.TargetMajorVersion {
		db.Close()
		return nil, fmt.Errorf(
			"version mismatch: plan database is PostgreSQL %d, but target database is PostgreSQL %d (exact major version match required)",
			majorVersion, config.TargetMajorVersion,
		)
	}

	// Generate temporary schema name with unique timestamp
	tempSchema := GenerateTempSchemaName()

	return &ExternalDatabase{
		db:                 db,
		host:               config.Host,
		port:               config.Port,
		database:           config.Database,
		username:           config.Username,
		password:           config.Password,
		tempSchema:         tempSchema,
		targetMajorVersion: config.TargetMajorVersion,
	}, nil
}

// GetConnectionDetails returns all connection details needed to connect to the external database
func (ed *ExternalDatabase) GetConnectionDetails() (host string, port int, database, username, password string) {
	return ed.host, ed.port, ed.database, ed.username, ed.password
}

// GetSchemaName returns the temporary schema name used for desired state validation
func (ed *ExternalDatabase) GetSchemaName() string {
	return ed.tempSchema
}

// ApplySchema creates a temporary schema and applies SQL to it.
// The temporary schema name includes a timestamp to avoid conflicts.
func (ed *ExternalDatabase) ApplySchema(ctx context.Context, schema string, sql string) error {
	// Note: We use the temporary schema name (ed.tempSchema) instead of the user-provided schema name
	// This ensures we don't interfere with existing schemas in the external database

	// Create the temporary schema
	createSchemaSQL := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS \"%s\"", ed.tempSchema)
	if _, err := ed.db.ExecContext(ctx, createSchemaSQL); err != nil {
		return fmt.Errorf("failed to create temporary schema %s: %w", ed.tempSchema, err)
	}

	// Set search_path to the temporary schema
	setSearchPathSQL := fmt.Sprintf("SET search_path TO \"%s\"", ed.tempSchema)
	if _, err := ed.db.ExecContext(ctx, setSearchPathSQL); err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}

	// Strip schema qualifications from SQL before applying to temporary schema
	// This ensures that objects are created in the temporary schema via search_path
	// rather than being explicitly qualified with the original schema name
	schemaAgnosticSQL := stripSchemaQualifications(sql, schema)

	// Execute the SQL directly
	// Note: Desired state SQL should never contain operations like CREATE INDEX CONCURRENTLY
	// that cannot run in transactions. Those are migration details, not state declarations.
	if _, err := ed.db.ExecContext(ctx, schemaAgnosticSQL); err != nil {
		return fmt.Errorf("failed to apply schema SQL to temporary schema %s: %w", ed.tempSchema, err)
	}

	return nil
}

// Stop closes the connection and drops the temporary schema (best effort).
// Errors during cleanup are logged but don't cause failures.
func (ed *ExternalDatabase) Stop() error {
	// Drop the temporary schema (best effort - don't fail if this errors)
	if ed.db != nil && ed.tempSchema != "" {
		ctx := context.Background()
		dropSchemaSQL := fmt.Sprintf("DROP SCHEMA IF EXISTS \"%s\" CASCADE", ed.tempSchema)
		// Ignore errors - this is best effort cleanup
		_, _ = ed.db.ExecContext(ctx, dropSchemaSQL)
	}

	// Close database connection
	if ed.db != nil {
		return ed.db.Close()
	}

	return nil
}

// detectMajorVersion queries the database to determine its PostgreSQL major version
func detectMajorVersion(db *sql.DB) (int, error) {
	ctx := context.Background()

	// Query PostgreSQL version number (e.g., 170005 for 17.5)
	var versionNum int
	err := db.QueryRowContext(ctx, "SHOW server_version_num").Scan(&versionNum)
	if err != nil {
		return 0, fmt.Errorf("failed to query PostgreSQL version: %w", err)
	}

	// Extract major version: version_num / 10000
	// e.g., 170005 / 10000 = 17
	majorVersion := versionNum / 10000

	return majorVersion, nil
}
