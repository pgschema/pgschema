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

// setupPostgresContainerWithDB creates a new PostgreSQL instance with custom database settings
func setupPostgresContainerWithDB(ctx context.Context, t *testing.T, database, username, password string) *ContainerInfo {
	// Extract test name and create unique runtime path
	testName := strings.ReplaceAll(t.Name(), "/", "_") // Replace slashes for subtest names
	timestamp := time.Now().Format("20060102_150405_999999")
	runtimePath := filepath.Join(os.TempDir(), fmt.Sprintf("pgschema-test-%s-%s", testName, timestamp))

	// Find an available port
	port, err := findAvailablePort()
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
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
		t.Fatalf("Failed to start embedded postgres: %v", err)
	}

	// Build connection string
	host := "localhost"
	testDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		username, password, host, port, database)

	// Connect to database
	conn, err := sql.Open("pgx", testDSN)
	if err != nil {
		postgres.Stop()
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Test the connection
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		postgres.Stop()
		t.Fatalf("Failed to ping database: %v", err)
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