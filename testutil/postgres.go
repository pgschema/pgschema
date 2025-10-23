// Package testutil provides shared test utilities for pgschema
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresVersion is an alias for the embedded-postgres version type
// This allows test files to reference the version type without directly importing embedded-postgres
type PostgresVersion = embeddedpostgres.PostgresVersion

// getPostgresVersion returns the PostgreSQL version to use for testing.
// It reads from the PGSCHEMA_POSTGRES_VERSION environment variable,
// defaulting to "17" if not set.
func getPostgresVersion() embeddedpostgres.PostgresVersion {
	versionStr := os.Getenv("PGSCHEMA_POSTGRES_VERSION")
	switch versionStr {
	case "14":
		return embeddedpostgres.PostgresVersion("14.18.0")
	case "15":
		return embeddedpostgres.PostgresVersion("15.13.0")
	case "16":
		return embeddedpostgres.PostgresVersion("16.9.0")
	case "17", "":
		return embeddedpostgres.PostgresVersion("17.5.0")
	default:
		return embeddedpostgres.PostgresVersion("17.5.0")
	}
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

// TestPostgres holds PostgreSQL instance connection details for testing
type TestPostgres struct {
	Database    *embeddedpostgres.EmbeddedPostgres
	Host        string
	Port        int
	DSN         string
	Conn        *sql.DB
	RuntimePath string
}

// SetupTestPostgres creates a new PostgreSQL test instance with standard credentials
func SetupTestPostgres(ctx context.Context, t *testing.T) *TestPostgres {
	// Standard test database credentials
	database := "testdb"
	username := "testuser"
	password := "testpass"

	// Extract test name and create unique runtime path
	testName := "shared"
	if t != nil {
		testName = strings.ReplaceAll(t.Name(), "/", "_") // Replace slashes for subtest names
	}
	timestamp := time.Now().Format("20060102_150405.000000000")
	runtimePath := filepath.Join(os.TempDir(), fmt.Sprintf("pgschema-test-%s-%s", testName, timestamp))

	// Find an available port
	port, err := findAvailablePort()
	if err != nil {
		if t != nil {
			t.Fatalf("Failed to find available port: %v", err)
		} else {
			panic(fmt.Sprintf("Failed to find available port: %v", err))
		}
	}

	// Configure embedded postgres with unique runtime path and dynamic port
	config := embeddedpostgres.DefaultConfig().
		Version(getPostgresVersion()).
		Database(database).
		Username(username).
		Password(password).
		Port(uint32(port)).
		RuntimePath(runtimePath).
		DataPath(filepath.Join(runtimePath, "data")).
		Logger(io.Discard). // Suppress embedded-postgres startup logs
		StartParameters(map[string]string{
			"logging_collector": "off",        // Disable log collector
			"log_destination":   "stderr",     // Send logs to stderr (which we discard above)
			"log_min_messages":  "PANIC",      // Only log PANIC level messages
			"log_statement":     "none",       // Don't log SQL statements
			"log_min_duration_statement": "-1", // Don't log slow queries
		})

	// Create and start PostgreSQL instance
	postgres := embeddedpostgres.NewDatabase(config)
	err = postgres.Start()
	if err != nil {
		if t != nil {
			t.Fatalf("Failed to start embedded postgres: %v", err)
		} else {
			panic(fmt.Sprintf("Failed to start embedded postgres: %v", err))
		}
	}

	// Build connection string
	host := "localhost"
	testDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		username, password, host, port, database)

	// Connect to database
	conn, err := sql.Open("pgx", testDSN)
	if err != nil {
		postgres.Stop()
		if t != nil {
			t.Fatalf("Failed to connect to database: %v", err)
		} else {
			panic(fmt.Sprintf("Failed to connect to database: %v", err))
		}
	}

	// Test the connection
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		postgres.Stop()
		if t != nil {
			t.Fatalf("Failed to ping database: %v", err)
		} else {
			panic(fmt.Sprintf("Failed to ping database: %v", err))
		}
	}

	return &TestPostgres{
		Database:    postgres,
		Host:        host,
		Port:        port,
		DSN:         testDSN,
		Conn:        conn,
		RuntimePath: runtimePath,
	}
}

// Terminate cleans up the database instance and connection
func (tp *TestPostgres) Terminate(ctx context.Context, t *testing.T) {
	tp.Conn.Close()
	if err := tp.Database.Stop(); err != nil {
		if t != nil {
			t.Logf("Failed to stop embedded postgres: %v", err)
		}
		// Silently ignore errors if t is nil (called from TestMain cleanup)
	}
	// Clean up the runtime directory
	if tp.RuntimePath != "" {
		if err := os.RemoveAll(tp.RuntimePath); err != nil {
			if t != nil {
				t.Logf("Failed to clean up runtime directory: %v", err)
			}
			// Silently ignore errors if t is nil
		}
	}
}

// SetEnvPassword sets the PGPASSWORD environment variable
func SetEnvPassword(password string) {
	os.Setenv("PGPASSWORD", password)
}

// TestConnectionConfig stores connection settings for save/restore operations
type TestConnectionConfig struct {
	Host   string
	Port   int
	DB     string
	User   string
	Schema string
}

// ============================================================================
// Version Detection and Mapping
// ============================================================================

// MapToEmbeddedPostgresVersion maps a PostgreSQL major version to embedded-postgres version
// Supported versions: 14, 15, 16, 17
func MapToEmbeddedPostgresVersion(majorVersion int) (embeddedpostgres.PostgresVersion, error) {
	switch majorVersion {
	case 14:
		return embeddedpostgres.PostgresVersion("14.18.0"), nil
	case 15:
		return embeddedpostgres.PostgresVersion("15.13.0"), nil
	case 16:
		return embeddedpostgres.PostgresVersion("16.9.0"), nil
	case 17:
		return embeddedpostgres.PostgresVersion("17.5.0"), nil
	default:
		return "", fmt.Errorf("unsupported PostgreSQL version %d (supported: 14, 15, 16, 17)", majorVersion)
	}
}

// DetectPostgresVersion queries the target database to determine its PostgreSQL version
// and returns the corresponding embedded-postgres version string
func DetectPostgresVersion(db *sql.DB) (embeddedpostgres.PostgresVersion, error) {
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
	return MapToEmbeddedPostgresVersion(majorVersion)
}

// ParseVersionString parses a PostgreSQL version string (e.g., "17.5") and returns major version
func ParseVersionString(versionStr string) (int, error) {
	// Handle various formats: "17.5", "17.5.0", "PostgreSQL 17.5", etc.
	// Extract the version number part
	versionStr = strings.TrimSpace(versionStr)

	// Remove "PostgreSQL " prefix if present
	versionStr = strings.TrimPrefix(versionStr, "PostgreSQL ")

	// Split by "." and take the first part (major version)
	parts := strings.Split(versionStr, ".")
	if len(parts) == 0 {
		return 0, fmt.Errorf("invalid version string: %s", versionStr)
	}

	majorVersion, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("failed to parse major version from %s: %w", versionStr, err)
	}

	return majorVersion, nil
}

// ============================================================================
// Production EmbeddedPostgres Wrapper
// ============================================================================

// EmbeddedPostgres manages a temporary embedded PostgreSQL instance
// This is used both for testing and for the plan command in production
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

// GetDB returns the database connection
func (ep *EmbeddedPostgres) GetDB() *sql.DB {
	return ep.db
}

// GetConnectionInfo returns connection details
func (ep *EmbeddedPostgres) GetConnectionInfo() (host string, port int, database string) {
	return ep.host, ep.port, ep.database
}

// GetCredentials returns the username and password for the embedded PostgreSQL instance
func (ep *EmbeddedPostgres) GetCredentials() (username string, password string) {
	return ep.username, ep.password
}

// ResetSchema drops and recreates a schema, clearing all objects
// This is useful for tests that want to reuse the same embedded postgres instance
func (ep *EmbeddedPostgres) ResetSchema(ctx context.Context, schema string) error {
	// Drop the schema if it exists (CASCADE to drop all objects)
	dropSchemaSQL := fmt.Sprintf("DROP SCHEMA IF EXISTS \"%s\" CASCADE", schema)
	if _, err := ep.db.ExecContext(ctx, dropSchemaSQL); err != nil {
		return fmt.Errorf("failed to drop schema %s: %w", schema, err)
	}

	// Recreate the schema
	createSchemaSQL := fmt.Sprintf("CREATE SCHEMA \"%s\"", schema)
	if _, err := ep.db.ExecContext(ctx, createSchemaSQL); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", schema, err)
	}

	return nil
}

// ApplySchemaSQL applies SQL schema to the embedded PostgreSQL database
func (ep *EmbeddedPostgres) ApplySchemaSQL(ctx context.Context, schema string, sql string) error {
	// Create the schema if it doesn't exist
	createSchemaSQL := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS \"%s\"", schema)
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

// SetupSharedEmbeddedPostgres creates a shared embedded PostgreSQL instance for test suites.
// This instance can be reused across multiple test cases to significantly improve test performance.
//
// Usage example:
//
//	func TestMain(m *testing.M) {
//	    // Create shared embedded postgres for all tests
//	    embeddedPG := testutil.SetupSharedEmbeddedPostgres(nil, embeddedpostgres.PostgresVersion("17.5.0"))
//	    defer embeddedPG.Stop()
//
//	    // Run tests
//	    code := m.Run()
//	    os.Exit(code)
//	}
func SetupSharedEmbeddedPostgres(t testing.TB, version embeddedpostgres.PostgresVersion) *EmbeddedPostgres {
	config := &EmbeddedPostgresConfig{
		Version:  version,
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
	}

	embeddedPG, err := StartEmbeddedPostgres(config)
	if err != nil {
		if t != nil {
			t.Fatalf("Failed to start shared embedded PostgreSQL: %v", err)
		} else {
			panic("Failed to start shared embedded PostgreSQL: " + err.Error())
		}
	}

	return embeddedPG
}

// DetectPostgresVersionFromDB connects to a database and detects its version
// This is a convenience function that opens a connection, detects the version, and closes it
func DetectPostgresVersionFromDB(host string, port int, database, user, password string) (embeddedpostgres.PostgresVersion, error) {
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
	return DetectPostgresVersion(db)
}

// ============================================================================
// Shared Test Postgres for IR Tests
// ============================================================================

// SetupSharedTestPostgres creates a shared embedded postgres instance for test packages.
// This significantly improves test performance by avoiding repeated postgres startup/shutdown.
// The returned instance should be stored by the caller and passed to ParseSQLForTest.
//
// Usage in test packages:
//
//	var sharedTestPostgres *testutil.TestPostgres
//
//	func TestMain(m *testing.M) {
//	    ctx := context.Background()
//	    sharedTestPostgres = testutil.SetupSharedTestPostgres(ctx, nil)
//	    defer sharedTestPostgres.Terminate(ctx, nil)
//
//	    code := m.Run()
//	    os.Exit(code)
//	}
func SetupSharedTestPostgres(ctx context.Context, t testing.TB) *TestPostgres {
	// Convert testing.TB to *testing.T if needed
	var tt *testing.T
	if t != nil {
		if tPtr, ok := t.(*testing.T); ok {
			tt = tPtr
		} else {
			// For testing.TB that's not *testing.T (like *testing.M), create a dummy *testing.T
			// This is safe because SetupTestPostgres only uses t for Fatalf on errors
			panic("SetupSharedTestPostgres requires *testing.T or nil")
		}
	}
	return SetupTestPostgres(ctx, tt)
}

// ParseSQLForTest is a test helper that converts SQL to an inspectable database state
// using embedded PostgreSQL. This replaces the old parser-based approach for tests.
//
// The caller must provide a TestPostgres instance (typically from SetupSharedTestPostgres).
// The schema will be reset (dropped and recreated) to ensure clean state between test calls.
//
// This function returns the database connection that can be inspected. The caller should NOT
// close this connection as it belongs to the test postgres instance.
//
// This ensures tests use the same code path as production (database inspection) rather than parsing.
func ParseSQLForTest(t *testing.T, testPG *TestPostgres, sqlContent string, schema string) *sql.DB {
	t.Helper()

	ctx := context.Background()
	conn := testPG.Conn

	// Drop and recreate schema for clean state
	dropSchema := fmt.Sprintf("DROP SCHEMA IF EXISTS \"%s\" CASCADE", schema)
	if _, err := conn.ExecContext(ctx, dropSchema); err != nil {
		t.Fatalf("Failed to drop schema: %v", err)
	}
	createSchema := fmt.Sprintf("CREATE SCHEMA \"%s\"", schema)
	if _, err := conn.ExecContext(ctx, createSchema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Set search_path to target schema
	setSearchPathSQL := fmt.Sprintf("SET search_path TO \"%s\"", schema)
	if _, err := conn.ExecContext(ctx, setSearchPathSQL); err != nil {
		t.Fatalf("Failed to set search_path: %v", err)
	}

	// Execute the SQL
	if _, err := conn.ExecContext(ctx, sqlContent); err != nil {
		t.Fatalf("Failed to apply SQL to embedded PostgreSQL: %v", err)
	}

	return conn
}
