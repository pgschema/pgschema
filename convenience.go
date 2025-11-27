package pgschema

import (
	"context"

	"github.com/pgschema/pgschema/internal/plan"
)

// DumpSchema is a convenience function to dump a database schema as a single SQL string.
func DumpSchema(ctx context.Context, dbConfig DatabaseConfig) (string, error) {
	client := NewClient(dbConfig)
	return client.Dump(ctx, DumpOptions{})
}

// DumpSchemaToFile is a convenience function to dump a database schema to a single file.
func DumpSchemaToFile(ctx context.Context, dbConfig DatabaseConfig, filePath string) error {
	client := NewClient(dbConfig)
	_, err := client.Dump(ctx, DumpOptions{
		File: filePath,
	})
	return err
}

// DumpSchemaMultiFile is a convenience function to dump a database schema to multiple files.
func DumpSchemaMultiFile(ctx context.Context, dbConfig DatabaseConfig, basePath string) error {
	client := NewClient(dbConfig)
	_, err := client.Dump(ctx, DumpOptions{
		MultiFile: true,
		File:      basePath,
	})
	return err
}

// GeneratePlan is a convenience function to generate a migration plan from a desired state file.
func GeneratePlan(ctx context.Context, dbConfig DatabaseConfig, desiredStateFile string) (*plan.Plan, error) {
	client := NewClient(dbConfig)
	return client.Plan(ctx, PlanOptions{
		File: desiredStateFile,
	})
}

// ApplySchemaFile is a convenience function to apply a desired state schema file directly.
// This generates a plan and applies it in one operation.
func ApplySchemaFile(ctx context.Context, dbConfig DatabaseConfig, desiredStateFile string, autoApprove bool) error {
	client := NewClient(dbConfig)
	return client.Apply(ctx, ApplyOptions{
		File:        desiredStateFile,
		AutoApprove: autoApprove,
	})
}

// ApplyPlan is a convenience function to apply a pre-generated migration plan.
func ApplyPlan(ctx context.Context, dbConfig DatabaseConfig, migrationPlan *plan.Plan, autoApprove bool) error {
	client := NewClient(dbConfig)
	return client.Apply(ctx, ApplyOptions{
		Plan:        migrationPlan,
		AutoApprove: autoApprove,
	})
}

// QuietApplySchemaFile is like ApplySchemaFile but suppresses all output except errors.
func QuietApplySchemaFile(ctx context.Context, dbConfig DatabaseConfig, desiredStateFile string) error {
	client := NewClient(dbConfig)
	return client.Apply(ctx, ApplyOptions{
		File:        desiredStateFile,
		AutoApprove: true,
		Quiet:       true,
	})
}

// QuietApplyPlan is like ApplyPlan but suppresses all output except errors.
func QuietApplyPlan(ctx context.Context, dbConfig DatabaseConfig, migrationPlan *plan.Plan) error {
	client := NewClient(dbConfig)
	return client.Apply(ctx, ApplyOptions{
		Plan:        migrationPlan,
		AutoApprove: true,
		Quiet:       true,
	})
}