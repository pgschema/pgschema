package util

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
	"github.com/pgschema/pgschema/internal/logger"
)

// EmbeddedPostgres manages a temporary embedded PostgreSQL instance
type EmbeddedPostgres struct {
	instance    *embeddedpostgres.EmbeddedPostgres
	db          *sql.DB
	version     embeddedpostgres.PostgresVersion
	host        string
	port        int
	database    string
	username    string
	password    string
	runtimePath string
}

// EmbeddedPostgresConfig holds configuration for starting embedded PostgreSQL
type EmbeddedPostgresConfig struct {
	Version  embeddedpostgres.PostgresVersion
	Database string
	Username string
	Password string
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

// StartEmbeddedPostgres starts a temporary embedded PostgreSQL instance
func StartEmbeddedPostgres(config *EmbeddedPostgresConfig) (*EmbeddedPostgres, error) {
	log := logger.Get()

	// Create unique runtime path with timestamp
	timestamp := time.Now().Format("20060102_150405_999999")
	runtimePath := filepath.Join(os.TempDir(), fmt.Sprintf("pgschema-plan-%s", timestamp))

	// Find an available port
	port, err := findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	log.Debug("Starting embedded PostgreSQL",
		"version", config.Version,
		"port", port,
		"database", config.Database,
		"runtime_path", runtimePath,
	)

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
			"logging_collector":          "off",   // Disable log collector
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

	log.Debug("Embedded PostgreSQL started successfully",
		"host", host,
		"port", port,
	)

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

// GetDB returns the database connection
func (ep *EmbeddedPostgres) GetDB() *sql.DB {
	return ep.db
}

// GetConnectionInfo returns connection details
func (ep *EmbeddedPostgres) GetConnectionInfo() (host string, port int, database string) {
	return ep.host, ep.port, ep.database
}

// ResetSchema drops and recreates a schema, clearing all objects
// This is useful for tests that want to reuse the same embedded postgres instance
func (ep *EmbeddedPostgres) ResetSchema(ctx context.Context, schema string) error {
	log := logger.Get()
	log.Debug("Resetting schema in embedded PostgreSQL",
		"schema", schema,
	)

	// Drop the schema if it exists (CASCADE to drop all objects)
	dropSchemaSQL := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", QuoteIdentifier(schema))
	if _, err := ep.db.ExecContext(ctx, dropSchemaSQL); err != nil {
		return fmt.Errorf("failed to drop schema %s: %w", schema, err)
	}

	// Recreate the schema
	createSchemaSQL := fmt.Sprintf("CREATE SCHEMA %s", QuoteIdentifier(schema))
	if _, err := ep.db.ExecContext(ctx, createSchemaSQL); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", schema, err)
	}

	log.Debug("Schema reset successfully")
	return nil
}

// ApplySchemaSQL applies SQL schema to the embedded PostgreSQL database
func (ep *EmbeddedPostgres) ApplySchemaSQL(ctx context.Context, schema string, sql string) error {
	log := logger.Get()
	log.Debug("Applying schema SQL to embedded PostgreSQL",
		"schema", schema,
		"sql_length", len(sql),
	)

	// Create the schema if it doesn't exist
	createSchemaSQL := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", QuoteIdentifier(schema))
	if _, err := ep.db.ExecContext(ctx, createSchemaSQL); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", schema, err)
	}

	// Set search_path to the target schema
	setSearchPathSQL := fmt.Sprintf("SET search_path TO %s", QuoteIdentifier(schema))
	if _, err := ep.db.ExecContext(ctx, setSearchPathSQL); err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}

	// Execute the SQL directly
	// Note: Desired state SQL should never contain operations like CREATE INDEX CONCURRENTLY
	// that cannot run in transactions. Those are migration details, not state declarations.
	if _, err := ep.db.ExecContext(ctx, sql); err != nil {
		return fmt.Errorf("failed to apply schema SQL: %w", err)
	}

	log.Debug("Schema SQL applied successfully")
	return nil
}

// Stop stops and cleans up the embedded PostgreSQL instance
func (ep *EmbeddedPostgres) Stop() error {
	log := logger.Get()
	log.Debug("Stopping embedded PostgreSQL",
		"runtime_path", ep.runtimePath,
	)

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
			log.Debug("Failed to clean up runtime directory",
				"path", ep.runtimePath,
				"error", err,
			)
			// Don't return error here - just log it
		}
	}

	if stopErr != nil {
		return fmt.Errorf("failed to stop embedded PostgreSQL: %w", stopErr)
	}

	log.Debug("Embedded PostgreSQL stopped and cleaned up")
	return nil
}

// QuoteIdentifier quotes a PostgreSQL identifier (schema, table, column name)
// This is a simple implementation - for production use, consider using pq.QuoteIdentifier
func QuoteIdentifier(identifier string) string {
	// For now, just use the IR package's quote function
	// In a production system, you might want to use a proper quoting library
	return fmt.Sprintf("\"%s\"", identifier)
}
