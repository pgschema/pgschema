package diff

import (
	"fmt"
	"sort"

	"github.com/pgschema/pgschema/internal/ir"
)

// generateCreatePoliciesSQL generates CREATE POLICY statements
func generateCreatePoliciesSQL(w *SQLWriter, policies []*ir.RLSPolicy, targetSchema string) {
	// Sort policies by name for consistent ordering
	sortedPolicies := make([]*ir.RLSPolicy, len(policies))
	copy(sortedPolicies, policies)
	sort.Slice(sortedPolicies, func(i, j int) bool {
		return sortedPolicies[i].Name < sortedPolicies[j].Name
	})

	for _, policy := range sortedPolicies {
		w.WriteDDLSeparator()
		sql := generatePolicySQL(policy, targetSchema)
		w.WriteStatementWithComment("POLICY", policy.Name, policy.Schema, "", sql, targetSchema)
	}
}

// generateDropPoliciesSQL generates DROP POLICY statements
func generateDropPoliciesSQL(w *SQLWriter, policies []*ir.RLSPolicy, targetSchema string) {
	// Sort policies by name for consistent ordering
	sortedPolicies := make([]*ir.RLSPolicy, len(policies))
	copy(sortedPolicies, policies)
	sort.Slice(sortedPolicies, func(i, j int) bool {
		return sortedPolicies[i].Name < sortedPolicies[j].Name
	})

	for _, policy := range sortedPolicies {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s;", policy.Name, policy.Table)
		w.WriteStatementWithComment("POLICY", policy.Name, policy.Schema, "", sql, targetSchema)
	}
}

// generateModifyPoliciesSQL generates ALTER POLICY statements
func generateModifyPoliciesSQL(w *SQLWriter, diffs []*PolicyDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()

		// Check if this policy needs to be recreated (DROP + CREATE)
		if needsRecreate(diff.Old, diff.New) {
			// Generate DROP statement
			dropSQL := fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s;", diff.Old.Name, diff.Old.Table)
			w.WriteStatementWithComment("POLICY", diff.Old.Name, diff.Old.Schema, "", dropSQL, targetSchema)
			w.WriteDDLSeparator()

			// Generate CREATE statement
			createSQL := generatePolicySQL(diff.New, targetSchema)
			w.WriteStatementWithComment("POLICY", diff.New.Name, diff.New.Schema, "", createSQL, targetSchema)
		} else {
			// Generate ALTER POLICY statement
			sql := generateAlterPolicySQL(diff.Old, diff.New, targetSchema)
			w.WriteStatementWithComment("POLICY", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
		}
	}
}

// generateRLSChangesSQL generates RLS enable/disable statements
func generateRLSChangesSQL(w *SQLWriter, changes []*RLSChange, targetSchema string) {
	for _, change := range changes {
		w.WriteDDLSeparator()
		var sql string
		if change.Enabled {
			sql = fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", change.Table.Name)
		} else {
			sql = fmt.Sprintf("ALTER TABLE %s DISABLE ROW LEVEL SECURITY;", change.Table.Name)
		}
		w.WriteStatementWithComment("TABLE", change.Table.Name, change.Table.Schema, "", sql, targetSchema)
	}
}

// generateRLSDisableChangesSQL generates RLS disable statements only
func generateRLSDisableChangesSQL(w *SQLWriter, changes []*RLSChange, targetSchema string) {
	for _, change := range changes {
		if !change.Enabled {
			w.WriteDDLSeparator()
			sql := fmt.Sprintf("ALTER TABLE %s DISABLE ROW LEVEL SECURITY;", change.Table.Name)
			w.WriteStatementWithComment("TABLE", change.Table.Name, change.Table.Schema, "", sql, targetSchema)
		}
	}
}

// generateRLSEnableChangesSQL generates RLS enable statements only
func generateRLSEnableChangesSQL(w *SQLWriter, changes []*RLSChange, targetSchema string) {
	for _, change := range changes {
		if change.Enabled {
			w.WriteDDLSeparator()
			sql := fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", change.Table.Name)
			w.WriteStatementWithComment("TABLE", change.Table.Name, change.Table.Schema, "", sql, targetSchema)
		}
	}
}

// generatePolicySQL generates CREATE POLICY statement
func generatePolicySQL(policy *ir.RLSPolicy, targetSchema string) string {
	// Only include table name without schema if it's in the target schema
	tableName := getTableNameWithSchema(policy.Schema, policy.Table, targetSchema)

	policyStmt := fmt.Sprintf("CREATE POLICY %s ON %s", policy.Name, tableName)

	// Add command type if specified
	if policy.Command != ir.PolicyCommandAll {
		policyStmt += fmt.Sprintf(" FOR %s", policy.Command)
	}

	// Add roles if specified
	if len(policy.Roles) > 0 {
		policyStmt += " TO "
		for i, role := range policy.Roles {
			if i > 0 {
				policyStmt += ", "
			}
			policyStmt += role
		}
	}

	// Add USING clause if present
	if policy.Using != "" {
		policyStmt += fmt.Sprintf(" USING (%s)", policy.Using)
	}

	// Add WITH CHECK clause if present
	if policy.WithCheck != "" {
		policyStmt += fmt.Sprintf(" WITH CHECK (%s)", policy.WithCheck)
	}

	return policyStmt + ";"
}

// generateAlterPolicySQL generates ALTER POLICY statement for changes that don't require recreation
func generateAlterPolicySQL(old, new *ir.RLSPolicy, targetSchema string) string {
	// Only include table name without schema if it's in the target schema
	tableName := getTableNameWithSchema(new.Schema, new.Table, targetSchema)

	// For role changes, we need to alter the TO clause
	if !roleListsEqual(old.Roles, new.Roles) {
		alterStmt := fmt.Sprintf("ALTER POLICY %s ON %s TO ", new.Name, tableName)
		for i, role := range new.Roles {
			if i > 0 {
				alterStmt += ", "
			}
			alterStmt += role
		}
		return alterStmt + ";"
	}

	// For USING clause changes
	if old.Using != new.Using {
		return fmt.Sprintf("ALTER POLICY %s ON %s USING (%s);", new.Name, tableName, new.Using)
	}

	// For WITH CHECK clause changes
	if old.WithCheck != new.WithCheck {
		return fmt.Sprintf("ALTER POLICY %s ON %s WITH CHECK (%s);", new.Name, tableName, new.WithCheck)
	}

	// This shouldn't happen as needsRecreate should catch other changes
	return fmt.Sprintf("-- Policy %s requires recreation", new.Name)
}

// roleListsEqual compares two role lists for equality
func roleListsEqual(oldRoles, newRoles []string) bool {
	if len(oldRoles) != len(newRoles) {
		return false
	}
	for i, role := range oldRoles {
		if newRoles[i] != role {
			return false
		}
	}
	return true
}

// generateTableRLS generates RLS enablement and policies for a specific table
func generateTableRLS(w *SQLWriter, table *ir.Table, targetSchema string) {
	// Generate ALTER TABLE ... ENABLE ROW LEVEL SECURITY if needed
	if table.RLSEnabled {
		w.WriteDDLSeparator()
		var fullTableName string
		if table.Schema == targetSchema {
			fullTableName = table.Name
		} else {
			fullTableName = fmt.Sprintf("%s.%s", table.Schema, table.Name)
		}
		sql := fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", fullTableName)
		w.WriteStatementWithComment("TABLE", table.Name, table.Schema, "", sql, "")
	}

	// Generate policies for this table
	// Get sorted policy names for consistent output
	policyNames := make([]string, 0, len(table.Policies))
	for policyName := range table.Policies {
		policyNames = append(policyNames, policyName)
	}
	sort.Strings(policyNames)

	for _, policyName := range policyNames {
		policy := table.Policies[policyName]

		w.WriteDDLSeparator()
		sql := generatePolicySQL(policy, targetSchema)
		w.WriteStatementWithComment("POLICY", policyName, table.Schema, "", sql, targetSchema)
	}
}

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
