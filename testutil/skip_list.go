// Package testutil provides shared test utilities for pgschema
package testutil

import (
	"strings"
	"testing"
)

// skipListPG14_15 defines test cases that should be skipped for PostgreSQL 14-15.
//
// Reason for skipping:
// PostgreSQL 14-15 use pg_get_viewdef() which returns table-qualified column names (e.g., employees.id),
// while PostgreSQL 16+ returns simplified column names (e.g., id). This is a non-consequential
// formatting difference that does not impact correctness, but causes test comparison failures.
var skipListPG14_15 = []string{
	// View tests - pg_get_viewdef() formatting differences
	"create_view/add_view",
	"create_view/alter_view",
	"create_view/drop_view",
	"dependency/table_to_view",

	// Materialized view tests - same pg_get_viewdef() issue
	"create_materialized_view/add_materialized_view",
	"create_materialized_view/alter_materialized_view",
	"create_materialized_view/drop_materialized_view",
	"dependency/table_to_materialized_view",

	// Online materialized view index tests - depend on materialized view definitions
	"online/add_materialized_view_index",
	"online/alter_materialized_view_index",

	// Comment tests - fingerprint includes view definitions
	"comment/add_index_comment",
	"comment/add_view_comment",

	// Index tests - fingerprint includes view definitions
	"create_index/drop_index",

	// Trigger tests - fingerprint includes view definitions
	"create_trigger/add_trigger",

	// Migration tests - include views and materialized views
	"migrate/v4",
	"migrate/v5",

	// Dump integration tests - contain views with formatting differences
	"TestDumpCommand_Employee",
	"TestDumpCommand_Issue82ViewLogicExpr",

	// Include integration test - uses materialized views
	"TestIncludeIntegration",
}

// skipListForVersion maps PostgreSQL major versions to their skip lists.
var skipListForVersion = map[int][]string{
	14: skipListPG14_15,
	15: skipListPG14_15,
}

// ShouldSkipTest checks if a test should be skipped for the given PostgreSQL major version.
// If the test should be skipped, it calls t.Skipf() which stops test execution.
//
// Test name format examples:
//   - "create_view_add_view" (from TestDiffFromFiles subtests - underscores separate all parts)
//   - "create_view/add_view" (skip list patterns - underscores in category, slash before test)
//   - "TestDumpCommand_Employee" (from dump tests - starts with Test)
//
// Matching uses exact string match with flexible slash/underscore handling:
// Pattern "create_view/add_view" matches test name "create_view_add_view" (underscores)
func ShouldSkipTest(t *testing.T, testName string, majorVersion int) {
	t.Helper()

	// Get skip patterns for this version
	skipPatterns, exists := skipListForVersion[majorVersion]
	if !exists {
		return // No skips defined for this version
	}

	// Check if test name matches any skip pattern (exact match)
	for _, pattern := range skipPatterns {
		// Convert pattern slashes to underscores to match test name format
		// e.g., "create_view/add_view" -> "create_view_add_view"
		patternNormalized := strings.ReplaceAll(pattern, "/", "_")

		if testName == patternNormalized || testName == pattern {
			t.Skipf("Skipping test %q on PostgreSQL %d due to pg_get_viewdef() formatting differences (non-consequential)", testName, majorVersion)
		}
	}
}
