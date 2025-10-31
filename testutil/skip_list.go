// Package testutil provides shared test utilities for pgschema
package testutil

import (
	"strings"
	"testing"
)

// skipListForVersion defines test cases that should be skipped for specific PostgreSQL versions.
// The key is the PostgreSQL major version, and the value is a list of test name patterns to skip.
//
// Reason for skipping:
// PostgreSQL 14-15 use pg_get_viewdef() which returns table-qualified column names (e.g., employees.id),
// while PostgreSQL 16+ returns simplified column names (e.g., id). This is a non-consequential
// formatting difference that does not impact correctness, but causes test comparison failures.
var skipListForVersion = map[int][]string{
	14: {
		// View tests - pg_get_viewdef() formatting differences
		"create_view/add_view",
		"create_view/alter_view",
		"create_view/drop_view",

		// Materialized view tests - same pg_get_viewdef() issue
		"create_materialized_view/add_materialized_view",
		"create_materialized_view/alter_materialized_view",
		"create_materialized_view/drop_materialized_view",

		// Online materialized view index tests - depend on materialized view definitions
		"online/add_materialized_view_index",
		"online/alter_materialized_view_index",

		// Comment tests - fingerprint includes view definitions
		"comment/add_index_comment",
		"comment/add_view_comment",

		// Index tests - fingerprint includes view definitions
		"create_index/drop_index",

		// Migration tests - include views and materialized views
		"migrate/v4",
		"migrate/v5",

		// Dump integration tests - contain views with formatting differences
		"TestDumpCommand_Employee",
		"TestDumpCommand_Issue82ViewLogicExpr",

		// Include integration test - uses materialized views
		"TestIncludeIntegration",
	},
	15: {
		// Same issues as PostgreSQL 14
		"create_view/add_view",
		"create_view/alter_view",
		"create_view/drop_view",
		"create_materialized_view/add_materialized_view",
		"create_materialized_view/alter_materialized_view",
		"create_materialized_view/drop_materialized_view",
		"online/add_materialized_view_index",
		"online/alter_materialized_view_index",
		"comment/add_index_comment",
		"comment/add_view_comment",
		"create_index/drop_index",
		"migrate/v4",
		"migrate/v5",
		"TestDumpCommand_Employee",
		"TestDumpCommand_Issue82ViewLogicExpr",
		"TestIncludeIntegration",
	},
}

// ShouldSkipTest checks if a test should be skipped for the given PostgreSQL major version.
// It returns true if the test name matches any pattern in the skip list for that version.
//
// Test name format examples:
//   - "create_view_add_view" (from TestDiffFromFiles subtests - underscores separate all parts)
//   - "create_view/add_view" (skip list patterns - underscores in category, slash before test)
//   - "TestDumpCommand_Employee" (from dump tests - starts with Test)
//
// Matching logic:
//   - For patterns without slashes: exact string match
//   - For patterns with slashes: flexible match allowing either underscores or slashes
func ShouldSkipTest(t *testing.T, testName string, majorVersion int) bool {
	t.Helper()

	// Get skip patterns for this version
	skipPatterns, exists := skipListForVersion[majorVersion]
	if !exists {
		return false // No skips defined for this version
	}

	// Check if test name matches any skip pattern
	for _, pattern := range skipPatterns {
		// If pattern contains a slash, do flexible matching
		// e.g., "create_view/add_view" should match "create_view_add_view"
		if strings.Contains(pattern, "/") {
			// Convert pattern to also check with underscores
			patternUnderscore := strings.ReplaceAll(pattern, "/", "_")
			if testName == patternUnderscore || strings.HasPrefix(testName, patternUnderscore+"_") {
				t.Skipf("Skipping test %q on PostgreSQL %d due to pg_get_viewdef() formatting differences (non-consequential)", testName, majorVersion)
				return true
			}
		} else {
			// Exact match for patterns without slashes (like "TestDumpCommand_Employee")
			if testName == pattern {
				t.Skipf("Skipping test %q on PostgreSQL %d due to pg_get_viewdef() formatting differences (non-consequential)", testName, majorVersion)
				return true
			}
		}
	}

	return false
}
