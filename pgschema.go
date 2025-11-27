// Package pgschema provides a programmatic API for PostgreSQL schema management.
// It offers Terraform-style declarative schema migration workflows with dump/plan/apply operations.
package pgschema

import (
	"context"
	"fmt"

	"github.com/pgschema/pgschema/cmd/apply"
	"github.com/pgschema/pgschema/cmd/dump"
	planCmd "github.com/pgschema/pgschema/cmd/plan"
	"github.com/pgschema/pgschema/internal/plan"
	"github.com/pgschema/pgschema/internal/postgres"
)

// DatabaseConfig holds connection details for a PostgreSQL database.
type DatabaseConfig struct {
	Host     string // Database server host
	Port     int    // Database server port
	Database string // Database name
	User     string // Database user
	Password string // Database password (optional)
	Schema   string // Target schema name (default: "public")
}

// PlanDatabaseConfig holds optional configuration for using an external database
// for plan generation instead of embedded PostgreSQL.
type PlanDatabaseConfig struct {
	Host     string // Plan database server host
	Port     int    // Plan database server port
	Database string // Plan database name
	User     string // Plan database user
	Password string // Plan database password (optional)
}

// DumpOptions configures how schema dumping is performed.
type DumpOptions struct {
	DatabaseConfig
	MultiFile bool   // Output to multiple files organized by object type
	File      string // Output file path (required when MultiFile is true)
}

// PlanOptions configures how migration planning is performed.
type PlanOptions struct {
	DatabaseConfig
	File              string                  // Path to desired state SQL schema file
	ApplicationName   string                  // Application name for database connection (default: "pgschema")
	PlanDatabase      *PlanDatabaseConfig     // Optional external database for plan generation
	DesiredStateProvider postgres.DesiredStateProvider // Optional pre-created provider
}

// ApplyOptions configures how migration application is performed.
type ApplyOptions struct {
	DatabaseConfig
	File            string                  // Path to desired state SQL schema file (alternative to Plan)
	Plan            *plan.Plan              // Pre-generated plan (alternative to File)
	AutoApprove     bool                    // Apply changes without prompting for approval
	NoColor         bool                    // Disable colored output
	Quiet           bool                    // Suppress plan display and progress messages
	LockTimeout     string                  // Maximum time to wait for database locks (e.g., "30s", "5m", "1h")
	ApplicationName string                  // Application name for database connection (default: "pgschema")
	PlanDatabase    *PlanDatabaseConfig     // Optional external database for plan generation (only used with File)
	DesiredStateProvider postgres.DesiredStateProvider // Optional pre-created provider (only used with File)
}

// Client provides the main interface for pgschema operations.
type Client struct {
	// Default configuration that can be overridden by individual operations
	defaultDB   DatabaseConfig
	defaultApp  string
}

// NewClient creates a new pgschema client with default database configuration.
func NewClient(dbConfig DatabaseConfig) *Client {
	// Set default schema if not provided
	if dbConfig.Schema == "" {
		dbConfig.Schema = "public"
	}

	return &Client{
		defaultDB:  dbConfig,
		defaultApp: "pgschema",
	}
}

// Dump extracts the current database schema and returns it as a SQL string.
// If opts.MultiFile is true, files are written to disk and an empty string is returned.
func (c *Client) Dump(ctx context.Context, opts DumpOptions) (string, error) {
	// Apply defaults
	if opts.Host == "" {
		opts.DatabaseConfig = c.defaultDB
	}
	if opts.Schema == "" {
		opts.Schema = "public"
	}

	// Convert to internal config
	config := &dump.DumpConfig{
		Host:      opts.Host,
		Port:      opts.Port,
		DB:        opts.Database,
		User:      opts.User,
		Password:  opts.Password,
		Schema:    opts.Schema,
		MultiFile: opts.MultiFile,
		File:      opts.File,
	}

	return dump.ExecuteDump(config)
}

// Plan generates a migration plan by comparing the current database state with a desired state.
// The desired state is defined by SQL files that will be applied to a temporary database
// (embedded PostgreSQL or external database) for validation.
func (c *Client) Plan(ctx context.Context, opts PlanOptions) (*plan.Plan, error) {
	// Apply defaults
	if opts.Host == "" {
		opts.DatabaseConfig = c.defaultDB
	}
	if opts.Schema == "" {
		opts.Schema = "public"
	}
	if opts.ApplicationName == "" {
		opts.ApplicationName = c.defaultApp
	}

	// Convert to internal config
	config := &planCmd.PlanConfig{
		Host:            opts.Host,
		Port:            opts.Port,
		DB:              opts.Database,
		User:            opts.User,
		Password:        opts.Password,
		Schema:          opts.Schema,
		File:            opts.File,
		ApplicationName: opts.ApplicationName,
	}

	// Add plan database config if provided
	if opts.PlanDatabase != nil {
		config.PlanDBHost = opts.PlanDatabase.Host
		config.PlanDBPort = opts.PlanDatabase.Port
		config.PlanDBDatabase = opts.PlanDatabase.Database
		config.PlanDBUser = opts.PlanDatabase.User
		config.PlanDBPassword = opts.PlanDatabase.Password
	}

	// Use provided provider or create a new one
	var provider postgres.DesiredStateProvider
	var err error

	if opts.DesiredStateProvider != nil {
		provider = opts.DesiredStateProvider
	} else {
		provider, err = planCmd.CreateDesiredStateProvider(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create desired state provider: %w", err)
		}
		defer provider.Stop()
	}

	return planCmd.GeneratePlan(config, provider)
}

// Apply executes a migration plan to update the database schema.
// You can either provide a pre-generated plan (opts.Plan) or a desired state file (opts.File).
func (c *Client) Apply(ctx context.Context, opts ApplyOptions) error {
	// Validate that either File or Plan is provided
	if opts.File == "" && opts.Plan == nil {
		return fmt.Errorf("either File or Plan must be provided")
	}

	// Apply defaults
	if opts.Host == "" {
		opts.DatabaseConfig = c.defaultDB
	}
	if opts.Schema == "" {
		opts.Schema = "public"
	}
	if opts.ApplicationName == "" {
		opts.ApplicationName = c.defaultApp
	}

	// Convert to internal config
	config := &apply.ApplyConfig{
		Host:            opts.Host,
		Port:            opts.Port,
		DB:              opts.Database,
		User:            opts.User,
		Password:        opts.Password,
		Schema:          opts.Schema,
		File:            opts.File,
		Plan:            opts.Plan,
		AutoApprove:     opts.AutoApprove,
		NoColor:         opts.NoColor,
		Quiet:           opts.Quiet,
		LockTimeout:     opts.LockTimeout,
		ApplicationName: opts.ApplicationName,
	}

	// Handle desired state provider for File mode
	var provider postgres.DesiredStateProvider
	var err error

	if opts.File != "" && opts.DesiredStateProvider == nil {
		// Need to create a provider for File mode
		planConfig := &planCmd.PlanConfig{
			Host:            opts.Host,
			Port:            opts.Port,
			DB:              opts.Database,
			User:            opts.User,
			Password:        opts.Password,
			Schema:          opts.Schema,
			File:            opts.File,
			ApplicationName: opts.ApplicationName,
		}

		// Add plan database config if provided
		if opts.PlanDatabase != nil {
			planConfig.PlanDBHost = opts.PlanDatabase.Host
			planConfig.PlanDBPort = opts.PlanDatabase.Port
			planConfig.PlanDBDatabase = opts.PlanDatabase.Database
			planConfig.PlanDBUser = opts.PlanDatabase.User
			planConfig.PlanDBPassword = opts.PlanDatabase.Password
		}

		provider, err = planCmd.CreateDesiredStateProvider(planConfig)
		if err != nil {
			return fmt.Errorf("failed to create desired state provider: %w", err)
		}
		defer provider.Stop()
	} else if opts.File != "" {
		provider = opts.DesiredStateProvider
	}

	return apply.ApplyMigration(config, provider)
}

// CreateEmbeddedProvider creates an embedded PostgreSQL provider for plan generation.
// The provider must be stopped by calling Stop() when done.
func CreateEmbeddedProvider(ctx context.Context, targetDB DatabaseConfig, pgVersion postgres.PostgresVersion) (*postgres.EmbeddedPostgres, error) {
	config := &planCmd.PlanConfig{
		Host:     targetDB.Host,
		Port:     targetDB.Port,
		DB:       targetDB.Database,
		User:     targetDB.User,
		Password: targetDB.Password,
		Schema:   targetDB.Schema,
	}

	return planCmd.CreateEmbeddedPostgresForPlan(config, pgVersion)
}

// CreateExternalProvider creates an external database provider for plan generation.
// The provider must be stopped by calling Stop() when done.
func CreateExternalProvider(ctx context.Context, targetDB DatabaseConfig, planDB PlanDatabaseConfig) (postgres.DesiredStateProvider, error) {
	// Detect target database version
	pgVersion, err := postgres.DetectPostgresVersionFromDB(
		targetDB.Host,
		targetDB.Port,
		targetDB.Database,
		targetDB.User,
		targetDB.Password,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to detect PostgreSQL version: %w", err)
	}

	// Extract major version
	var targetMajorVersion int
	_, err = fmt.Sscanf(string(pgVersion), "%d.", &targetMajorVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL version %s: %w", pgVersion, err)
	}

	externalConfig := &postgres.ExternalDatabaseConfig{
		Host:               planDB.Host,
		Port:               planDB.Port,
		Database:           planDB.Database,
		Username:           planDB.User,
		Password:           planDB.Password,
		TargetMajorVersion: targetMajorVersion,
	}

	return postgres.NewExternalDatabase(externalConfig)
}

// DetectPostgresVersion detects the PostgreSQL version of a database.
func DetectPostgresVersion(ctx context.Context, dbConfig DatabaseConfig) (postgres.PostgresVersion, error) {
	return postgres.DetectPostgresVersionFromDB(
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Database,
		dbConfig.User,
		dbConfig.Password,
	)
}