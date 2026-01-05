package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/ir"
)

// generateCreateDefaultPrivilegesSQL generates ALTER DEFAULT PRIVILEGES GRANT statements
func generateCreateDefaultPrivilegesSQL(privileges []*ir.DefaultPrivilege, targetSchema string, collector *diffCollector) {
	for _, dp := range privileges {
		sql := generateGrantDefaultPrivilegeSQL(dp, targetSchema)

		context := &diffContext{
			Type:                DiffTypeDefaultPrivilege,
			Operation:           DiffOperationCreate,
			Path:                fmt.Sprintf("default_privileges.%s.%s", dp.ObjectType, dp.Grantee),
			Source:              dp,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateDropDefaultPrivilegesSQL generates ALTER DEFAULT PRIVILEGES REVOKE statements
func generateDropDefaultPrivilegesSQL(privileges []*ir.DefaultPrivilege, targetSchema string, collector *diffCollector) {
	for _, dp := range privileges {
		sql := generateRevokeDefaultPrivilegeSQL(dp, targetSchema)

		context := &diffContext{
			Type:                DiffTypeDefaultPrivilege,
			Operation:           DiffOperationDrop,
			Path:                fmt.Sprintf("default_privileges.%s.%s", dp.ObjectType, dp.Grantee),
			Source:              dp,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateModifyDefaultPrivilegesSQL generates ALTER DEFAULT PRIVILEGES statements for modifications
func generateModifyDefaultPrivilegesSQL(diffs []*defaultPrivilegeDiff, targetSchema string, collector *diffCollector) {
	for _, diff := range diffs {
		statements := diff.generateAlterDefaultPrivilegeStatements(targetSchema)
		for _, stmt := range statements {
			context := &diffContext{
				Type:                DiffTypeDefaultPrivilege,
				Operation:           DiffOperationAlter,
				Path:                fmt.Sprintf("default_privileges.%s.%s", diff.New.ObjectType, diff.New.Grantee),
				Source:              diff,
				CanRunInTransaction: true,
			}

			collector.collect(context, stmt)
		}
	}
}

// generateGrantDefaultPrivilegeSQL generates ALTER DEFAULT PRIVILEGES GRANT statement
func generateGrantDefaultPrivilegeSQL(dp *ir.DefaultPrivilege, targetSchema string) string {
	// Sort privileges for deterministic output
	sortedPrivs := make([]string, len(dp.Privileges))
	copy(sortedPrivs, dp.Privileges)
	sort.Strings(sortedPrivs)

	privStr := strings.Join(sortedPrivs, ", ")
	grantee := dp.Grantee
	if grantee == "" || grantee == "PUBLIC" {
		// PUBLIC is a special keyword meaning "all roles", not an identifier
		grantee = "PUBLIC"
	} else {
		grantee = ir.QuoteIdentifier(grantee)
	}

	sql := fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT %s ON %s TO %s",
		ir.QuoteIdentifier(targetSchema), privStr, dp.ObjectType, grantee)

	if dp.WithGrantOption {
		sql += " WITH GRANT OPTION"
	}

	return sql + ";"
}

// generateRevokeDefaultPrivilegeSQL generates ALTER DEFAULT PRIVILEGES REVOKE statement
func generateRevokeDefaultPrivilegeSQL(dp *ir.DefaultPrivilege, targetSchema string) string {
	// Sort privileges for deterministic output
	sortedPrivs := make([]string, len(dp.Privileges))
	copy(sortedPrivs, dp.Privileges)
	sort.Strings(sortedPrivs)

	privStr := strings.Join(sortedPrivs, ", ")
	grantee := dp.Grantee
	if grantee == "" || grantee == "PUBLIC" {
		// PUBLIC is a special keyword meaning "all roles", not an identifier
		grantee = "PUBLIC"
	} else {
		grantee = ir.QuoteIdentifier(grantee)
	}

	return fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s REVOKE %s ON %s FROM %s;",
		ir.QuoteIdentifier(targetSchema), privStr, dp.ObjectType, grantee)
}

// generateAlterDefaultPrivilegeStatements generates statements for privilege modifications
func (d *defaultPrivilegeDiff) generateAlterDefaultPrivilegeStatements(targetSchema string) []string {
	var statements []string

	// Find privileges to revoke (in old but not in new)
	oldPrivSet := make(map[string]bool)
	for _, p := range d.Old.Privileges {
		oldPrivSet[p] = true
	}
	newPrivSet := make(map[string]bool)
	for _, p := range d.New.Privileges {
		newPrivSet[p] = true
	}

	var toRevoke []string
	for p := range oldPrivSet {
		if !newPrivSet[p] {
			toRevoke = append(toRevoke, p)
		}
	}

	var toGrant []string
	for p := range newPrivSet {
		if !oldPrivSet[p] {
			toGrant = append(toGrant, p)
		}
	}

	grantee := d.New.Grantee
	if grantee == "" || grantee == "PUBLIC" {
		// PUBLIC is a special keyword meaning "all roles", not an identifier
		grantee = "PUBLIC"
	} else {
		grantee = ir.QuoteIdentifier(grantee)
	}
	quotedSchema := ir.QuoteIdentifier(targetSchema)

	// Generate REVOKE for removed privileges
	if len(toRevoke) > 0 {
		sort.Strings(toRevoke)
		statements = append(statements, fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s REVOKE %s ON %s FROM %s;",
			quotedSchema, strings.Join(toRevoke, ", "), d.Old.ObjectType, grantee))
	}

	// Generate GRANT for added privileges
	if len(toGrant) > 0 {
		sort.Strings(toGrant)
		sql := fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT %s ON %s TO %s",
			quotedSchema, strings.Join(toGrant, ", "), d.New.ObjectType, grantee)
		if d.New.WithGrantOption {
			sql += " WITH GRANT OPTION"
		}
		statements = append(statements, sql+";")
	}

	// Handle WITH GRANT OPTION changes (if privileges are same but grant option changed)
	if len(toRevoke) == 0 && len(toGrant) == 0 && d.Old.WithGrantOption != d.New.WithGrantOption {
		// Need to revoke and re-grant with new option
		sortedPrivs := make([]string, len(d.New.Privileges))
		copy(sortedPrivs, d.New.Privileges)
		sort.Strings(sortedPrivs)
		privStr := strings.Join(sortedPrivs, ", ")

		// Revoke first
		statements = append(statements, fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s REVOKE %s ON %s FROM %s;",
			quotedSchema, privStr, d.New.ObjectType, grantee))

		// Then grant with correct option
		sql := fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT %s ON %s TO %s",
			quotedSchema, privStr, d.New.ObjectType, grantee)
		if d.New.WithGrantOption {
			sql += " WITH GRANT OPTION"
		}
		statements = append(statements, sql+";")
	}

	return statements
}

// GetObjectName returns a unique identifier for the default privilege diff
func (d *defaultPrivilegeDiff) GetObjectName() string {
	return string(d.New.ObjectType) + ":" + d.New.Grantee
}

// defaultPrivilegesEqual checks if two default privileges are structurally equal
func defaultPrivilegesEqual(old, new *ir.DefaultPrivilege) bool {
	if old.ObjectType != new.ObjectType {
		return false
	}
	if old.Grantee != new.Grantee {
		return false
	}
	if old.WithGrantOption != new.WithGrantOption {
		return false
	}

	// Compare privileges (order-independent)
	if len(old.Privileges) != len(new.Privileges) {
		return false
	}

	oldPrivSet := make(map[string]bool)
	for _, p := range old.Privileges {
		oldPrivSet[p] = true
	}
	for _, p := range new.Privileges {
		if !oldPrivSet[p] {
			return false
		}
	}

	return true
}
