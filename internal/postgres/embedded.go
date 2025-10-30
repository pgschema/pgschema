// Package postgres provides embedded PostgreSQL functionality for production use.
// This package is used by the plan command to create temporary PostgreSQL instances
// for validating desired state schemas.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresVersion is an alias for the embedded-postgres version type.
type PostgresVersion = embeddedpostgres.PostgresVersion

// EmbeddedPostgres manages a temporary embedded PostgreSQL instance.
// This is used by the plan command to validate desired state schemas.
type EmbeddedPostgres struct {
	instance    *embeddedpostgres.EmbeddedPostgres
	db          *sql.DB
	version     PostgresVersion
	host        string
	port        int
	database    string
	username    string
	password    string
	runtimePath string
}

// EmbeddedPostgresConfig holds configuration for starting embedded PostgreSQL
type EmbeddedPostgresConfig struct {
	Version  PostgresVersion
	Database string
	Username string
	Password string
}

// DetectPostgresVersionFromDB connects to a database and detects its version
// This is a convenience function that opens a connection, detects the version, and closes it
func DetectPostgresVersionFromDB(host string, port int, database, user, password string) (PostgresVersion, error) {
	// Build connection string
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=prefer",
		user, password, host, port, database)

	// Connect to database
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test the connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		return "", fmt.Errorf("failed to ping database: %w", err)
	}

	// Detect version
	return detectPostgresVersion(db)
}

// StartEmbeddedPostgres starts a temporary embedded PostgreSQL instance
func StartEmbeddedPostgres(config *EmbeddedPostgresConfig) (*EmbeddedPostgres, error) {
	// Create unique runtime path with timestamp (using nanoseconds for uniqueness)
	timestamp := time.Now().Format("20060102_150405.000000000")
	runtimePath := filepath.Join(os.TempDir(), fmt.Sprintf("pgschema-plan-%s", timestamp))

	// Find an available port
	port, err := findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// Configure embedded postgres
	pgConfig := embeddedpostgres.DefaultConfig().
		Version(config.Version).
		Database(config.Database).
		Username(config.Username).
		Password(config.Password).
		Port(uint32(port)).
		RuntimePath(runtimePath).
		DataPath(filepath.Join(runtimePath, "data")).
		Logger(io.Discard). // Suppress embedded-postgres startup logs
		StartParameters(map[string]string{
			"logging_collector":          "off",    // Disable log collector
			"log_destination":            "stderr", // Send logs to stderr (which we discard)
			"log_min_messages":           "PANIC",  // Only log PANIC level messages
			"log_statement":              "none",   // Don't log SQL statements
			"log_min_duration_statement": "-1",     // Don't log slow queries
		})

	// Create and start PostgreSQL instance
	instance := embeddedpostgres.NewDatabase(pgConfig)
	if err := instance.Start(); err != nil {
		return nil, fmt.Errorf("failed to start embedded PostgreSQL: %w", err)
	}

	// Build connection string
	host := "localhost"
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.Username, config.Password, host, port, config.Database)

	// Connect to database
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		instance.Stop()
		os.RemoveAll(runtimePath)
		return nil, fmt.Errorf("failed to connect to embedded PostgreSQL: %w", err)
	}

	// Test the connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		instance.Stop()
		os.RemoveAll(runtimePath)
		return nil, fmt.Errorf("failed to ping embedded PostgreSQL: %w", err)
	}

	return &EmbeddedPostgres{
		instance:    instance,
		db:          db,
		version:     config.Version,
		host:        host,
		port:        port,
		database:    config.Database,
		username:    config.Username,
		password:    config.Password,
		runtimePath: runtimePath,
	}, nil
}

// Stop stops and cleans up the embedded PostgreSQL instance
func (ep *EmbeddedPostgres) Stop() error {
	// Close database connection
	if ep.db != nil {
		ep.db.Close()
	}

	// Stop PostgreSQL instance
	var stopErr error
	if ep.instance != nil {
		stopErr = ep.instance.Stop()
	}

	// Clean up runtime directory
	if ep.runtimePath != "" {
		if err := os.RemoveAll(ep.runtimePath); err != nil {
			// Don't return error here - just ignore cleanup failures
			// This can happen on Windows when files are still in use
		}
	}

	if stopErr != nil {
		return fmt.Errorf("failed to stop embedded PostgreSQL: %w", stopErr)
	}

	return nil
}

// GetConnectionDetails returns all connection details needed to connect to the embedded PostgreSQL instance
func (ep *EmbeddedPostgres) GetConnectionDetails() (host string, port int, database, username, password string) {
	return ep.host, ep.port, ep.database, ep.username, ep.password
}

// GetSchemaName returns the schema name to inspect.
// For embedded postgres, this is managed externally, so we return empty string
// and rely on the caller to track the schema name.
func (ep *EmbeddedPostgres) GetSchemaName() string {
	// Embedded postgres doesn't track schema name - it's provided by the caller in ApplySchema
	// The caller (GeneratePlan) needs to use config.Schema for inspection
	return ""
}

// ApplySchema resets a schema (drops and recreates it) and applies SQL to it.
// This ensures a clean state before applying the desired schema definition.
func (ep *EmbeddedPostgres) ApplySchema(ctx context.Context, schema string, sql string) error {
	// Drop the schema if it exists (CASCADE to drop all objects)
	dropSchemaSQL := fmt.Sprintf("DROP SCHEMA IF EXISTS \"%s\" CASCADE", schema)
	if _, err := ep.db.ExecContext(ctx, dropSchemaSQL); err != nil {
		return fmt.Errorf("failed to drop schema %s: %w", schema, err)
	}

	// Create the schema
	createSchemaSQL := fmt.Sprintf("CREATE SCHEMA \"%s\"", schema)
	if _, err := ep.db.ExecContext(ctx, createSchemaSQL); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", schema, err)
	}

	// Set search_path to the target schema
	setSearchPathSQL := fmt.Sprintf("SET search_path TO \"%s\"", schema)
	if _, err := ep.db.ExecContext(ctx, setSearchPathSQL); err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}

	// Execute the SQL directly
	// Note: Desired state SQL should never contain operations like CREATE INDEX CONCURRENTLY
	// that cannot run in transactions. Those are migration details, not state declarations.
	if _, err := ep.db.ExecContext(ctx, sql); err != nil {
		return fmt.Errorf("failed to apply schema SQL: %w", err)
	}

	return nil
}

// findAvailablePort finds an available TCP port for PostgreSQL to use
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// mapToEmbeddedPostgresVersion maps a PostgreSQL major version to embedded-postgres version
// Supported versions: 14, 15, 16, 17
func mapToEmbeddedPostgresVersion(majorVersion int) (PostgresVersion, error) {
	switch majorVersion {
	case 14:
		return PostgresVersion("14.18.0"), nil
	case 15:
		return PostgresVersion("15.13.0"), nil
	case 16:
		return PostgresVersion("16.9.0"), nil
	case 17:
		return PostgresVersion("17.5.0"), nil
	default:
		return "", fmt.Errorf("unsupported PostgreSQL version %d (supported: 14, 15, 16, 17)", majorVersion)
	}
}

// detectPostgresVersion queries the target database to determine its PostgreSQL version
// and returns the corresponding embedded-postgres version string
func detectPostgresVersion(db *sql.DB) (PostgresVersion, error) {
	ctx := context.Background()

	// Query PostgreSQL version number (e.g., 170005 for 17.5)
	var versionNum int
	err := db.QueryRowContext(ctx, "SHOW server_version_num").Scan(&versionNum)
	if err != nil {
		return "", fmt.Errorf("failed to query PostgreSQL version: %w", err)
	}

	// Extract major version: version_num / 10000
	// e.g., 170005 / 10000 = 17
	majorVersion := versionNum / 10000

	// Map to embedded-postgres version
	return mapToEmbeddedPostgresVersion(majorVersion)
}
