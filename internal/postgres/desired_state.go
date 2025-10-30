// Package postgres provides PostgreSQL functionality for desired state management.
// This file defines the interface for desired state providers (embedded or external databases).
package postgres

import "context"

// DesiredStateProvider is an interface that abstracts the desired state database provider.
// It can be implemented by either embedded PostgreSQL or an external database connection.
type DesiredStateProvider interface {
	// GetConnectionDetails returns connection details for IR inspection
	// Returns: host, port, database, username, password
	GetConnectionDetails() (string, int, string, string, string)

	// GetSchemaName returns the actual schema name to inspect.
	// For embedded postgres: returns the user-provided schema name
	// For external database: returns the temporary schema name with timestamp
	GetSchemaName() string

	// ApplySchema applies the desired state SQL to a schema.
	// For embedded postgres: resets the schema (drop/recreate)
	// For external database: creates temporary schema with timestamp suffix
	ApplySchema(ctx context.Context, schema string, sql string) error

	// Stop performs cleanup.
	// For embedded postgres: stops instance and removes temp directory
	// For external database: drops temporary schema (best effort) and closes connection
	Stop() error
}
