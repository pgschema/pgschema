package util

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

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
	return mapToEmbeddedPostgresVersion(majorVersion)
}

// DetectPostgresVersionFromConfig queries the target database using connection config
// and returns the corresponding embedded-postgres version string
func DetectPostgresVersionFromConfig(config *ConnectionConfig) (embeddedpostgres.PostgresVersion, error) {
	// Connect to target database
	db, err := Connect(config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to detect version: %w", err)
	}
	defer db.Close()

	return DetectPostgresVersion(db)
}

// mapToEmbeddedPostgresVersion maps a PostgreSQL major version to embedded-postgres version
// Supported versions: 14, 15, 16, 17
func mapToEmbeddedPostgresVersion(majorVersion int) (embeddedpostgres.PostgresVersion, error) {
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
