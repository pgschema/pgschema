package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// policiesEqual compares two policies for equality
func policiesEqual(old, new *ir.RLSPolicy) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Table != new.Table {
		return false
	}
	if old.Name != new.Name {
		return false
	}
	if old.Command != new.Command {
		return false
	}
	if old.Permissive != new.Permissive {
		return false
	}
	if old.Using != new.Using {
		return false
	}
	if old.WithCheck != new.WithCheck {
		return false
	}
	if len(old.Roles) != len(new.Roles) {
		return false
	}
	for i, role := range old.Roles {
		if new.Roles[i] != role {
			return false
		}
	}
	return true
}

// needsRecreate determines if a policy change requires DROP/CREATE instead of ALTER
func needsRecreate(old, new *ir.RLSPolicy) bool {
	// Name changes require recreation (we don't use ALTER POLICY RENAME)
	if old.Name != new.Name {
		return true
	}
	// Command changes require recreation
	if old.Command != new.Command {
		return true
	}
	// Permissive/Restrictive changes require recreation
	if old.Permissive != new.Permissive {
		return true
	}
	// All other changes (roles, using, with_check) can use ALTER POLICY
	return false
}

// GenerateDropPolicySQL generates SQL for dropping policies
func GenerateDropPolicySQL(policies []*ir.RLSPolicy) []string {
	var statements []string
	
	// Sort policies by schema.table.name for consistent ordering
	sortedPolicies := make([]*ir.RLSPolicy, len(policies))
	copy(sortedPolicies, policies)
	sort.Slice(sortedPolicies, func(i, j int) bool {
		keyI := sortedPolicies[i].Schema + "." + sortedPolicies[i].Table + "." + sortedPolicies[i].Name
		keyJ := sortedPolicies[j].Schema + "." + sortedPolicies[j].Table + "." + sortedPolicies[j].Name
		return keyI < keyJ
	})
	
	for _, policy := range sortedPolicies {
		tableName := getTableNameWithSchema(policy.Schema, policy.Table, policy.Schema)
		statements = append(statements, fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s;", policy.Name, tableName))
	}
	
	return statements
}

// GenerateCreatePolicySQL generates SQL for creating policies
func GenerateCreatePolicySQL(policies []*ir.RLSPolicy) []string {
	var statements []string
	
	// Sort policies by schema.table.name for consistent ordering
	sortedPolicies := make([]*ir.RLSPolicy, len(policies))
	copy(sortedPolicies, policies)
	sort.Slice(sortedPolicies, func(i, j int) bool {
		keyI := sortedPolicies[i].Schema + "." + sortedPolicies[i].Table + "." + sortedPolicies[i].Name
		keyJ := sortedPolicies[j].Schema + "." + sortedPolicies[j].Table + "." + sortedPolicies[j].Name
		return keyI < keyJ
	})
	
	for _, policy := range sortedPolicies {
		stmt := generateCreatePolicySQL(policy)
		statements = append(statements, stmt)
	}
	
	return statements
}

// GenerateAlterPolicySQL generates SQL for modifying policies
func GenerateAlterPolicySQL(policyDiffs []*PolicyDiff) []string {
	var statements []string
	
	// Sort modified policies by schema.table.name for consistent ordering
	sortedPolicyDiffs := make([]*PolicyDiff, len(policyDiffs))
	copy(sortedPolicyDiffs, policyDiffs)
	sort.Slice(sortedPolicyDiffs, func(i, j int) bool {
		keyI := sortedPolicyDiffs[i].New.Schema + "." + sortedPolicyDiffs[i].New.Table + "." + sortedPolicyDiffs[i].New.Name
		keyJ := sortedPolicyDiffs[j].New.Schema + "." + sortedPolicyDiffs[j].New.Table + "." + sortedPolicyDiffs[j].New.Name
		return keyI < keyJ
	})
	
	for _, policyDiff := range sortedPolicyDiffs {
		if needsRecreate(policyDiff.Old, policyDiff.New) {
			// DROP and CREATE for cases that can't be ALTERed
			tableName := getTableNameWithSchema(policyDiff.Old.Schema, policyDiff.Old.Table, policyDiff.Old.Schema)
			statements = append(statements, fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s;", policyDiff.Old.Name, tableName))
			statements = append(statements, generateCreatePolicySQL(policyDiff.New))
		} else {
			// Use ALTER POLICY for supported changes
			stmts := generateAlterPolicySQL(policyDiff.Old, policyDiff.New)
			statements = append(statements, stmts...)
		}
	}
	
	return statements
}

// GenerateRLSChangeSQL generates SQL for enabling/disabling RLS
func GenerateRLSChangeSQL(rlsChanges []*RLSChange) []string {
	var statements []string
	
	// Sort RLS changes by schema.table for consistent ordering
	sortedChanges := make([]*RLSChange, len(rlsChanges))
	copy(sortedChanges, rlsChanges)
	sort.Slice(sortedChanges, func(i, j int) bool {
		keyI := sortedChanges[i].Table.Schema + "." + sortedChanges[i].Table.Name
		keyJ := sortedChanges[j].Table.Schema + "." + sortedChanges[j].Table.Name
		return keyI < keyJ
	})
	
	for _, change := range sortedChanges {
		tableName := getTableNameWithSchema(change.Table.Schema, change.Table.Name, change.Table.Schema)
		if change.Enabled {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", tableName))
		} else {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s DISABLE ROW LEVEL SECURITY;", tableName))
		}
	}
	
	return statements
}

// generateCreatePolicySQL generates CREATE POLICY SQL for a policy
func generateCreatePolicySQL(policy *ir.RLSPolicy) string {
	var stmt strings.Builder
	
	tableName := getTableNameWithSchema(policy.Schema, policy.Table, policy.Schema)
	stmt.WriteString(fmt.Sprintf("CREATE POLICY %s ON %s", policy.Name, tableName))
	
	// Add AS clause (PERMISSIVE is default, only specify if RESTRICTIVE)
	if !policy.Permissive {
		stmt.WriteString(" AS RESTRICTIVE")
	}
	
	// Add FOR clause (only if not ALL, matching original IR behavior)
	if policy.Command != ir.PolicyCommandAll {
		stmt.WriteString(fmt.Sprintf(" FOR %s", policy.Command))
	}
	
	// Add TO clause (roles)
	if len(policy.Roles) > 0 {
		stmt.WriteString(" TO ")
		for i, role := range policy.Roles {
			if i > 0 {
				stmt.WriteString(", ")
			}
			stmt.WriteString(role)
		}
	}
	
	// Add USING clause
	if policy.Using != "" {
		stmt.WriteString(fmt.Sprintf(" USING (%s)", policy.Using))
	}
	
	// Add WITH CHECK clause
	if policy.WithCheck != "" {
		stmt.WriteString(fmt.Sprintf(" WITH CHECK (%s)", policy.WithCheck))
	}
	
	stmt.WriteString(";")
	return stmt.String()
}

// generateAlterPolicySQL generates ALTER POLICY SQL for changes that can be altered
func generateAlterPolicySQL(old, new *ir.RLSPolicy) []string {
	var statements []string
	
	tableName := getTableNameWithSchema(old.Schema, old.Table, old.Schema)
	
	// Handle role changes
	if len(old.Roles) != len(new.Roles) || !slicesEqual(old.Roles, new.Roles) {
		stmt := fmt.Sprintf("ALTER POLICY %s ON %s TO ", old.Name, tableName)
		if len(new.Roles) > 0 {
			stmt += strings.Join(new.Roles, ", ")
		} else {
			stmt += "PUBLIC"
		}
		stmt += ";"
		statements = append(statements, stmt)
	}
	
	// Handle USING expression changes
	if old.Using != new.Using {
		if new.Using != "" {
			statements = append(statements, fmt.Sprintf("ALTER POLICY %s ON %s USING (%s);", old.Name, tableName, new.Using))
		}
	}
	
	// Handle WITH CHECK expression changes
	if old.WithCheck != new.WithCheck {
		if new.WithCheck != "" {
			statements = append(statements, fmt.Sprintf("ALTER POLICY %s ON %s WITH CHECK (%s);", old.Name, tableName, new.WithCheck))
		}
	}
	
	return statements
}

// slicesEqual compares two string slices for equality
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if b[i] != v {
			return false
		}
	}
	return true
}