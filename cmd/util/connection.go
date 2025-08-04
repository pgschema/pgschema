package util

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pgschema/pgschema/internal/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// ConnectionConfig holds database connection parameters
type ConnectionConfig struct {
	Host            string
	Port            int
	Database        string
	User            string
	Password        string
	SSLMode         string
	ApplicationName string
}

// Connect establishes a database connection using the provided configuration
func Connect(config *ConnectionConfig) (*sql.DB, error) {
	log := logger.Get()
	
	log.Debug("Attempting database connection",
		"host", config.Host,
		"port", config.Port,
		"database", config.Database,
		"user", config.User,
		"sslmode", config.SSLMode,
		"application_name", config.ApplicationName,
	)
	
	dsn := buildDSN(config)
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Debug("Database connection failed", "error", err)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Test the connection
	if err := conn.Ping(); err != nil {
		log.Debug("Database ping failed", "error", err)
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	log.Debug("Database connection established successfully")
	return conn, nil
}

// buildDSN constructs a PostgreSQL connection string from connection parameters
func buildDSN(config *ConnectionConfig) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("host=%s", config.Host))
	parts = append(parts, fmt.Sprintf("port=%d", config.Port))
	parts = append(parts, fmt.Sprintf("dbname=%s", config.Database))
	parts = append(parts, fmt.Sprintf("user=%s", config.User))

	if config.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", config.Password))
	}

	if config.SSLMode != "" {
		parts = append(parts, fmt.Sprintf("sslmode=%s", config.SSLMode))
	}

	if config.ApplicationName != "" {
		parts = append(parts, fmt.Sprintf("application_name=%s", config.ApplicationName))
	}

	return strings.Join(parts, " ")
}
