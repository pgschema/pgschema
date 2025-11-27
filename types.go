package pgschema

import (
	"github.com/pgschema/pgschema/internal/plan"
	"github.com/pgschema/pgschema/internal/postgres"
	"github.com/pgschema/pgschema/ir"
)

// Re-export important types for external consumption

// Plan represents a migration plan that can be applied to a database.
type Plan = plan.Plan

// DesiredStateProvider abstracts the desired state database provider.
// It can be implemented by either embedded PostgreSQL or an external database connection.
type DesiredStateProvider = postgres.DesiredStateProvider

// EmbeddedPostgres represents an embedded PostgreSQL instance for plan generation.
type EmbeddedPostgres = postgres.EmbeddedPostgres

// PostgresVersion represents a PostgreSQL version string.
type PostgresVersion = postgres.PostgresVersion

// ExternalDatabaseConfig holds configuration for using an external database for plan generation.
type ExternalDatabaseConfig = postgres.ExternalDatabaseConfig

// EmbeddedPostgresConfig holds configuration for starting embedded PostgreSQL.
type EmbeddedPostgresConfig = postgres.EmbeddedPostgresConfig

// IR represents the intermediate representation of a database schema.
type IR = ir.IR

// Schema represents a database schema with all its objects.
type Schema = ir.Schema

// Table represents a database table with its columns, constraints, indexes, etc.
type Table = ir.Table

// Column represents a table column.
type Column = ir.Column

// Constraint represents a table constraint (primary key, foreign key, etc.).
type Constraint = ir.Constraint

// Index represents a database index.
type Index = ir.Index

// View represents a database view (regular or materialized).
type View = ir.View

// Function represents a database function.
type Function = ir.Function

// Procedure represents a database procedure.
type Procedure = ir.Procedure

// Sequence represents a database sequence.
type Sequence = ir.Sequence

// Type represents a custom database type (enum, composite, domain).
type Type = ir.Type

// Trigger represents a database trigger.
type Trigger = ir.Trigger

// RLSPolicy represents a row-level security policy.
type RLSPolicy = ir.RLSPolicy

// Aggregate represents a custom aggregate function.
type Aggregate = ir.Aggregate

// IgnoreConfig represents configuration for ignoring objects during operations.
type IgnoreConfig = ir.IgnoreConfig

