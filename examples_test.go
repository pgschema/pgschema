package pgschema_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pgschema/pgschema/pgschema"
)

// ExampleDumpSchema demonstrates how to dump a database schema as a SQL string.
func ExampleDumpSchema() {
	ctx := context.Background()

	dbConfig := pgschema.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "myapp",
		User:     "postgres",
		Password: "password",
		Schema:   "public",
	}

	schema, err := pgschema.DumpSchema(ctx, dbConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Schema dump:")
	fmt.Println(schema)
}

// ExampleDumpSchemaToFile demonstrates how to dump a database schema to a file.
func ExampleDumpSchemaToFile() {
	ctx := context.Background()

	dbConfig := pgschema.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "myapp",
		User:     "postgres",
		Password: "password",
		Schema:   "public",
	}

	err := pgschema.DumpSchemaToFile(ctx, dbConfig, "schema.sql")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Schema dumped to schema.sql")
}

// ExampleGeneratePlan demonstrates how to generate a migration plan.
func ExampleGeneratePlan() {
	ctx := context.Background()

	dbConfig := pgschema.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "myapp",
		User:     "postgres",
		Password: "password",
		Schema:   "public",
	}

	plan, err := pgschema.GeneratePlan(ctx, dbConfig, "desired_schema.sql")
	if err != nil {
		log.Fatal(err)
	}

	// Display the plan in human-readable format
	fmt.Println("Migration plan:")
	fmt.Println(plan.HumanColored(true))

	// You can also get JSON representation
	jsonPlan, err := plan.ToJSONWithDebug(false)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("JSON plan:", jsonPlan)
}

// ExampleApplySchemaFile demonstrates how to apply a schema file directly.
func ExampleApplySchemaFile() {
	ctx := context.Background()

	dbConfig := pgschema.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "myapp",
		User:     "postgres",
		Password: "password",
		Schema:   "public",
	}

	// Apply with user confirmation
	err := pgschema.ApplySchemaFile(ctx, dbConfig, "desired_schema.sql", false)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Schema applied successfully")
}

// ExampleApplySchemaFileAutoApprove demonstrates how to apply a schema file without confirmation.
func ExampleApplySchemaFileAutoApprove() {
	ctx := context.Background()

	dbConfig := pgschema.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "myapp",
		User:     "postgres",
		Password: "password",
		Schema:   "public",
	}

	// Apply automatically without user confirmation
	err := pgschema.ApplySchemaFile(ctx, dbConfig, "desired_schema.sql", true)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Schema applied successfully")
}

// ExampleQuietApplySchemaFile demonstrates how to apply a schema file silently.
func ExampleQuietApplySchemaFile() {
	ctx := context.Background()

	dbConfig := pgschema.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "myapp",
		User:     "postgres",
		Password: "password",
		Schema:   "public",
	}

	// Apply silently without any output
	err := pgschema.QuietApplySchemaFile(ctx, dbConfig, "desired_schema.sql")
	if err != nil {
		log.Fatal(err)
	}

	// No output - useful for automation
}

// ExampleClient demonstrates using the Client API for more control.
func ExampleClient() {
	ctx := context.Background()

	dbConfig := pgschema.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "myapp",
		User:     "postgres",
		Password: "password",
		Schema:   "public",
	}

	client := pgschema.NewClient(dbConfig)

	// Dump schema with custom options
	schema, err := client.Dump(ctx, pgschema.DumpOptions{
		MultiFile: true,
		File:      "schemas/",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Generate plan with custom application name
	plan, err := client.Plan(ctx, pgschema.PlanOptions{
		File:            "desired_schema.sql",
		ApplicationName: "my-migration-tool",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Apply with custom settings
	err = client.Apply(ctx, pgschema.ApplyOptions{
		Plan:        plan,
		AutoApprove: true,
		LockTimeout: "30s",
		NoColor:     true,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Migration completed")
	_ = schema // Silence unused variable
}

// ExampleExternalPlanDatabase demonstrates using an external database for plan generation.
func ExampleExternalPlanDatabase() {
	ctx := context.Background()

	dbConfig := pgschema.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "myapp",
		User:     "postgres",
		Password: "password",
		Schema:   "public",
	}

	planDBConfig := pgschema.PlanDatabaseConfig{
		Host:     "plan-db-host",
		Port:     5432,
		Database: "plan_validation_db",
		User:     "plan_user",
		Password: "plan_password",
	}

	client := pgschema.NewClient(dbConfig)

	// Generate plan using external database instead of embedded PostgreSQL
	plan, err := client.Plan(ctx, pgschema.PlanOptions{
		File:         "desired_schema.sql",
		PlanDatabase: &planDBConfig,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Generated plan using external database:")
	fmt.Println(plan.HumanColored(true))
}

// ExampleCreateProviders demonstrates how to create and manage providers manually.
func ExampleCreateProviders() {
	ctx := context.Background()

	dbConfig := pgschema.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "myapp",
		User:     "postgres",
		Password: "password",
		Schema:   "public",
	}

	// Detect PostgreSQL version
	pgVersion, err := pgschema.DetectPostgresVersion(ctx, dbConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Create embedded provider manually
	provider, err := pgschema.CreateEmbeddedProvider(ctx, dbConfig, pgVersion)
	if err != nil {
		log.Fatal(err)
	}
	defer provider.Stop() // Important: always stop the provider

	// Use the provider for plan generation
	client := pgschema.NewClient(dbConfig)
	plan, err := client.Plan(ctx, pgschema.PlanOptions{
		File:                 "desired_schema.sql",
		DesiredStateProvider: provider,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Plan generated with manually managed provider:")
	fmt.Println(plan.HumanColored(true))
}

// ExampleEnvironmentConfiguration demonstrates using environment variables for configuration.
func ExampleEnvironmentConfiguration() {
	ctx := context.Background()

	// Set environment variables
	os.Setenv("PGHOST", "localhost")
	os.Setenv("PGPORT", "5432")
	os.Setenv("PGDATABASE", "myapp")
	os.Setenv("PGUSER", "postgres")
	os.Setenv("PGPASSWORD", "password")

	// Configuration can be minimal when using environment variables
	dbConfig := pgschema.DatabaseConfig{
		Host:     os.Getenv("PGHOST"),
		Port:     5432, // Could also parse PGPORT
		Database: os.Getenv("PGDATABASE"),
		User:     os.Getenv("PGUSER"),
		Password: os.Getenv("PGPASSWORD"),
		Schema:   "public",
	}

	// Or use the convenience functions directly
	schema, err := pgschema.DumpSchema(ctx, dbConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Dumped %d characters of schema\n", len(schema))
}

// ExampleWorkflowComplete demonstrates a complete dump/edit/plan/apply workflow.
func ExampleWorkflowComplete() {
	ctx := context.Background()

	dbConfig := pgschema.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "myapp",
		User:     "postgres",
		Password: "password",
		Schema:   "public",
	}

	// Step 1: Dump current schema
	fmt.Println("Step 1: Dumping current schema...")
	err := pgschema.DumpSchemaToFile(ctx, dbConfig, "current_schema.sql")
	if err != nil {
		log.Fatal(err)
	}

	// Step 2: Edit the schema file (this would be done manually)
	// For this example, we'll assume the file is edited and saved as "desired_schema.sql"

	// Step 3: Generate migration plan
	fmt.Println("Step 3: Generating migration plan...")
	plan, err := pgschema.GeneratePlan(ctx, dbConfig, "desired_schema.sql")
	if err != nil {
		log.Fatal(err)
	}

	// Display the plan
	fmt.Println("Migration plan:")
	fmt.Println(plan.HumanColored(true))

	// Step 4: Apply the migration (in this example, we'll use auto-approve for simplicity)
	fmt.Println("Step 4: Applying migration...")
	err = pgschema.ApplyPlan(ctx, dbConfig, plan, true)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Migration workflow completed successfully!")
}