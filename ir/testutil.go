// Package ir provides an intermediate representation for PostgreSQL schemas
package ir

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// getPostgresVersion returns the PostgreSQL version to use for testing.
// It reads from the PGSCHEMA_POSTGRES_VERSION environment variable,
// defaulting to "17" if not set.
// Returns an error if an unsupported version is specified.
func getPostgresVersion() (embeddedpostgres.PostgresVersion, error) {
	versionStr := os.Getenv("PGSCHEMA_POSTGRES_VERSION")
	if versionStr == "" {
		return embeddedpostgres.PostgresVersion("17.5.0"), nil
	}

	switch versionStr {
	case "14":
		return embeddedpostgres.PostgresVersion("14.18.0"), nil
	case "15":
		return embeddedpostgres.PostgresVersion("15.13.0"), nil
	case "16":
		return embeddedpostgres.PostgresVersion("16.9.0"), nil
	case "17":
		return embeddedpostgres.PostgresVersion("17.5.0"), nil
	default:
		return "", fmt.Errorf("unsupported PGSCHEMA_POSTGRES_VERSION: %s (supported versions: 14, 15, 16, 17)", versionStr)
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

// ContainerInfo holds PostgreSQL instance connection details for testing
type ContainerInfo struct {
	Database    *embeddedpostgres.EmbeddedPostgres
	Host        string
	Port        int
	DSN         string
	Conn        *sql.DB
	RuntimePath string
}

// setupPostgresContainer creates a new PostgreSQL test container
func setupPostgresContainer(ctx context.Context, t *testing.T) *ContainerInfo {
	return setupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
}

// fatalError handles test failures - uses t.Fatalf if available, otherwise panics
func fatalError(t *testing.T, format string, args ...interface{}) {
	if t != nil {
		t.Fatalf(format, args...)
	} else {
		panic(fmt.Sprintf(format, args...))
	}
}

// setupPostgresContainerWithDB creates a new PostgreSQL instance with custom database settings
func setupPostgresContainerWithDB(ctx context.Context, t *testing.T, database, username, password string) *ContainerInfo {
	// Extract test name and create unique runtime path
	testName := "shared"
	if t != nil {
		testName = strings.ReplaceAll(t.Name(), "/", "_") // Replace slashes for subtest names
	}
	timestamp := time.Now().Format("20060102_150405_999999")
	runtimePath := filepath.Join(os.TempDir(), fmt.Sprintf("pgschema-test-%s-%s", testName, timestamp))

	// Find an available port
	port, err := findAvailablePort()
	if err != nil {
		fatalError(t, "Failed to find available port: %v", err)
	}

	// Get PostgreSQL version
	pgVersion, err := getPostgresVersion()
	if err != nil {
		fatalError(t, "Failed to get PostgreSQL version: %v", err)
	}

	// Configure embedded postgres with unique runtime path and dynamic port
	config := embeddedpostgres.DefaultConfig().
		Version(pgVersion).
		Database(database).
		Username(username).
		Password(password).
		Port(uint32(port)).
		RuntimePath(runtimePath).
		DataPath(filepath.Join(runtimePath, "data")).
		Logger(io.Discard). // Suppress embedded-postgres startup logs
		StartParameters(map[string]string{
			"logging_collector":          "off",    // Disable log collector
			"log_destination":            "stderr", // Send logs to stderr (which we discard above)
			"log_min_messages":           "PANIC",  // Only log PANIC level messages
			"log_statement":              "none",   // Don't log SQL statements
			"log_min_duration_statement": "-1",     // Don't log slow queries
		})

	// Create and start PostgreSQL instance
	postgres := embeddedpostgres.NewDatabase(config)
	err = postgres.Start()
	if err != nil {
		fatalError(t, "Failed to start embedded postgres: %v", err)
	}

	// Build connection string
	host := "localhost"
	testDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		username, password, host, port, database)

	// Connect to database
	conn, err := sql.Open("pgx", testDSN)
	if err != nil {
		postgres.Stop()
		fatalError(t, "Failed to connect to database: %v", err)
	}

	// Test the connection
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		postgres.Stop()
		fatalError(t, "Failed to ping database: %v", err)
	}

	return &ContainerInfo{
		Database:    postgres,
		Host:        host,
		Port:        port,
		DSN:         testDSN,
		Conn:        conn,
		RuntimePath: runtimePath,
	}
}

// terminate cleans up the database instance and connection
func (ci *ContainerInfo) terminate(ctx context.Context, t *testing.T) {
	ci.Conn.Close()
	if err := ci.Database.Stop(); err != nil {
		t.Logf("Failed to stop embedded postgres: %v", err)
	}
	// Clean up the runtime directory
	if ci.RuntimePath != "" {
		if err := os.RemoveAll(ci.RuntimePath); err != nil {
			t.Logf("Failed to clean up runtime directory: %v", err)
		}
	}
}

// sharedTestContainer holds an optional shared embedded postgres instance for tests
var sharedTestContainer *ContainerInfo

// SetSharedTestContainer sets a shared embedded postgres instance for ParseSQLForTest to reuse.
// This significantly improves test performance by avoiding repeated postgres startup/shutdown.
//
// Usage in test packages:
//
//	func TestMain(m *testing.M) {
//	    ctx := context.Background()
//	    container := ir.SetupSharedTestContainer(ctx, nil)
//	    defer container.Terminate(ctx, nil)
//
//	    code := m.Run()
//	    os.Exit(code)
//	}
func SetupSharedTestContainer(ctx context.Context, t testing.TB) *ContainerInfo {
	// Convert testing.TB to *testing.T if needed
	var tt *testing.T
	if t != nil {
		if tPtr, ok := t.(*testing.T); ok {
			tt = tPtr
		} else {
			// For testing.TB that's not *testing.T (like *testing.M), create a dummy *testing.T
			// This is safe because setupPostgresContainer only uses t for Fatalf on errors
			panic("SetupSharedTestContainer requires *testing.T or nil")
		}
	}
	container := setupPostgresContainer(ctx, tt)
	sharedTestContainer = container
	return container
}

// Terminate cleans up the container (exported for use by test packages)
func (ci *ContainerInfo) Terminate(ctx context.Context, t testing.TB) {
	// Convert testing.TB to *testing.T if needed
	var tt *testing.T
	if t != nil {
		if tPtr, ok := t.(*testing.T); ok {
			tt = tPtr
		}
		// For nil or other types, tt remains nil which is fine for terminate
	}
	ci.terminate(ctx, tt)
}

// ParseSQLForTest is a test helper that converts SQL to IR using embedded PostgreSQL.
// This replaces the old parser-based approach for tests.
//
// If a shared test container has been set via SetupSharedTestContainer, it will be reused
// (with the schema reset between calls). Otherwise, a new temporary instance is created.
//
// This ensures tests use the same code path as production (database inspection) rather than parsing.
func ParseSQLForTest(t *testing.T, sqlContent string, schema string) *IR {
	t.Helper()

	ctx := context.Background()

	var conn *sql.DB
	var needsCleanup bool

	if sharedTestContainer != nil {
		// Reuse shared container - reset the schema for clean state
		conn = sharedTestContainer.Conn
		needsCleanup = false

		// Drop and recreate schema
		dropSchema := fmt.Sprintf("DROP SCHEMA IF EXISTS \"%s\" CASCADE", schema)
		if _, err := conn.ExecContext(ctx, dropSchema); err != nil {
			t.Fatalf("Failed to drop schema: %v", err)
		}
		createSchema := fmt.Sprintf("CREATE SCHEMA \"%s\"", schema)
		if _, err := conn.ExecContext(ctx, createSchema); err != nil {
			t.Fatalf("Failed to create schema: %v", err)
		}
	} else {
		// Create new container for this test
		container := setupPostgresContainer(ctx, t)
		defer container.terminate(ctx, t)
		conn = container.Conn
		needsCleanup = true

		// Create schema if not public
		if schema != "public" {
			createSchemaSQL := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS \"%s\"", schema)
			if _, err := conn.ExecContext(ctx, createSchemaSQL); err != nil {
				t.Fatalf("Failed to create schema: %v", err)
			}
		}
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

	// Inspect the database to get IR
	inspector := NewInspector(conn, nil)
	ir, err := inspector.BuildIR(ctx, schema)
	if err != nil {
		t.Fatalf("Failed to inspect embedded PostgreSQL: %v", err)
	}

	// If we created a container just for this test, cleanup happens via defer above
	_ = needsCleanup

	return ir
}