// Package testutil provides shared test utilities for pgschema
package testutil

import (
	"context"
	"database/sql"
	"io"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var suppressedLogger = log.New(io.Discard, "", 0)

// getPostgresVersion returns the PostgreSQL version to use for testing.
// It reads from the PGSCHEMA_POSTGRES_VERSION environment variable,
// defaulting to "17" if not set.
func getPostgresVersion() string {
	if version := os.Getenv("PGSCHEMA_POSTGRES_VERSION"); version != "" {
		return version
	}
	return "17"
}

// ContainerInfo holds PostgreSQL container connection details
type ContainerInfo struct {
	Container testcontainers.Container
	Host      string
	Port      int
	DSN       string
	Conn      *sql.DB
}

// SetupPostgresContainer creates a new PostgreSQL test container
func SetupPostgresContainer(ctx context.Context, t *testing.T) *ContainerInfo {
	return SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
}

// SetupPostgresContainerWithDB creates a new PostgreSQL test container with custom database settings
func SetupPostgresContainerWithDB(ctx context.Context, t *testing.T, database, username, password string) *ContainerInfo {

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:"+getPostgresVersion()+"-alpine",
		postgres.WithDatabase(database),
		postgres.WithUsername(username),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
		testcontainers.WithLogger(suppressedLogger),
	)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	// Get connection string
	testDSN, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect to database
	conn, err := sql.Open("pgx", testDSN)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Get container connection details
	containerHost, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}
	containerPort, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	return &ContainerInfo{
		Container: postgresContainer,
		Host:      containerHost,
		Port:      containerPort.Int(),
		DSN:       testDSN,
		Conn:      conn,
	}
}

// Terminate cleans up the container and connection
func (ci *ContainerInfo) Terminate(ctx context.Context, t *testing.T) {
	ci.Conn.Close()
	if err := ci.Container.Terminate(ctx); err != nil {
		t.Logf("Failed to terminate container: %v", err)
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
