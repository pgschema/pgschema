// Package postgres provides embedded PostgreSQL functionality for testing only.
// This is a minimal wrapper around embedded-postgres used only in test files.
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

// EmbeddedPostgres manages a temporary embedded PostgreSQL instance for testing.
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

// StartEmbeddedPostgres starts a temporary embedded PostgreSQL instance for testing
func StartEmbeddedPostgres(config *EmbeddedPostgresConfig) (*EmbeddedPostgres, error) {
	// Create unique runtime path with timestamp
	timestamp := time.Now().Format("20060102_150405.000000000")
	runtimePath := filepath.Join(os.TempDir(), fmt.Sprintf("pgschema-test-%s", timestamp))

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
		Logger(io.Discard). // Suppress logs
		StartParameters(map[string]string{
			"logging_collector": "off",
			"log_destination":   "stderr",
			"log_min_messages":  "PANIC",
			"log_statement":     "none",
		})

	// Create and start PostgreSQL instance
	instance := embeddedpostgres.NewDatabase(pgConfig)
	if err := instance.Start(); err != nil {
		return nil, fmt.Errorf("failed to start embedded PostgreSQL: %w", err)
	}

	// Connect to database
	host := "localhost"
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.Username, config.Password, host, port, config.Database)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		instance.Stop()
		os.RemoveAll(runtimePath)
		return nil, fmt.Errorf("failed to connect to embedded PostgreSQL: %w", err)
	}

	// Test connection
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
	if ep.db != nil {
		ep.db.Close()
	}

	var stopErr error
	if ep.instance != nil {
		stopErr = ep.instance.Stop()
	}

	if ep.runtimePath != "" {
		os.RemoveAll(ep.runtimePath)
	}

	return stopErr
}

// GetConnectionDetails returns connection details for the embedded PostgreSQL instance
func (ep *EmbeddedPostgres) GetConnectionDetails() (host string, port int, database, username, password string) {
	return ep.host, ep.port, ep.database, ep.username, ep.password
}

// GetDB returns the database connection
func (ep *EmbeddedPostgres) GetDB() *sql.DB {
	return ep.db
}

// ApplySchema applies SQL to a schema (for testing)
func (ep *EmbeddedPostgres) ApplySchema(ctx context.Context, schema string, sql string) error {
	// Drop and recreate schema
	dropSchemaSQL := fmt.Sprintf("DROP SCHEMA IF EXISTS \"%s\" CASCADE", schema)
	if _, err := ep.db.ExecContext(ctx, dropSchemaSQL); err != nil {
		return fmt.Errorf("failed to drop schema %s: %w", schema, err)
	}

	createSchemaSQL := fmt.Sprintf("CREATE SCHEMA \"%s\"", schema)
	if _, err := ep.db.ExecContext(ctx, createSchemaSQL); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", schema, err)
	}

	// Set search_path
	setSearchPathSQL := fmt.Sprintf("SET search_path TO \"%s\"", schema)
	if _, err := ep.db.ExecContext(ctx, setSearchPathSQL); err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}

	// Execute SQL
	if _, err := ep.db.ExecContext(ctx, sql); err != nil {
		return fmt.Errorf("failed to apply schema SQL: %w", err)
	}

	return nil
}

// findAvailablePort finds an available TCP port
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// DetectPostgresVersionFromDB connects to a database and detects its version
func DetectPostgresVersionFromDB(host string, port int, database, user, password string) (PostgresVersion, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=prefer",
		user, password, host, port, database)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		return "", fmt.Errorf("failed to ping database: %w", err)
	}

	// Query version
	var versionNum int
	err = db.QueryRowContext(ctx, "SHOW server_version_num").Scan(&versionNum)
	if err != nil {
		return "", fmt.Errorf("failed to query PostgreSQL version: %w", err)
	}

	// Map to embedded-postgres version
	majorVersion := versionNum / 10000
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
		return "", fmt.Errorf("unsupported PostgreSQL version %d", majorVersion)
	}
}