package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/ir"
)

// generateCreatePrivilegesSQL generates GRANT statements for new privileges
func generateCreatePrivilegesSQL(privileges []*ir.Privilege, targetSchema string, collector *diffCollector) {
	for _, p := range privileges {
		sql := generateGrantPrivilegeSQL(p)

		context := &diffContext{
			Type:                DiffTypePrivilege,
			Operation:           DiffOperationCreate,
			Path:                fmt.Sprintf("privileges.%s.%s.%s", p.ObjectType, p.ObjectName, p.Grantee),
			Source:              p,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateDropPrivilegesSQL generates REVOKE statements for removed privileges
func generateDropPrivilegesSQL(privileges []*ir.Privilege, targetSchema string, collector *diffCollector) {
	for _, p := range privileges {
		sql := generateRevokePrivilegeSQL(p)

		context := &diffContext{
			Type:                DiffTypePrivilege,
			Operation:           DiffOperationDrop,
			Path:                fmt.Sprintf("privileges.%s.%s.%s", p.ObjectType, p.ObjectName, p.Grantee),
			Source:              p,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateModifyPrivilegesSQL generates ALTER privilege statements for modifications
func generateModifyPrivilegesSQL(diffs []*privilegeDiff, targetSchema string, collector *diffCollector) {
	for _, diff := range diffs {
		statements := diff.generateAlterPrivilegeStatements()
		for _, stmt := range statements {
			context := &diffContext{
				Type:                DiffTypePrivilege,
				Operation:           DiffOperationAlter,
				Path:                fmt.Sprintf("privileges.%s.%s.%s", diff.New.ObjectType, diff.New.ObjectName, diff.New.Grantee),
				Source:              diff,
				CanRunInTransaction: true,
			}

			collector.collect(context, stmt)
		}
	}
}

// generateGrantPrivilegeSQL generates a GRANT statement
func generateGrantPrivilegeSQL(p *ir.Privilege) string {
	// Sort privileges for deterministic output
	sortedPrivs := make([]string, len(p.Privileges))
	copy(sortedPrivs, p.Privileges)
	sort.Strings(sortedPrivs)

	privStr := strings.Join(sortedPrivs, ", ")
	grantee := formatGrantee(p.Grantee)
	objectRef := formatObjectReference(p.ObjectType, p.ObjectName)

	sql := fmt.Sprintf("GRANT %s ON %s TO %s", privStr, objectRef, grantee)

	if p.WithGrantOption {
		sql += " WITH GRANT OPTION"
	}

	return sql + ";"
}

// generateRevokePrivilegeSQL generates a REVOKE statement
func generateRevokePrivilegeSQL(p *ir.Privilege) string {
	// Sort privileges for deterministic output
	sortedPrivs := make([]string, len(p.Privileges))
	copy(sortedPrivs, p.Privileges)
	sort.Strings(sortedPrivs)

	privStr := strings.Join(sortedPrivs, ", ")
	grantee := formatGrantee(p.Grantee)
	objectRef := formatObjectReference(p.ObjectType, p.ObjectName)

	return fmt.Sprintf("REVOKE %s ON %s FROM %s;", privStr, objectRef, grantee)
}

// generateAlterPrivilegeStatements generates statements for privilege modifications
func (d *privilegeDiff) generateAlterPrivilegeStatements() []string {
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

	grantee := formatGrantee(d.New.Grantee)
	objectRef := formatObjectReference(d.New.ObjectType, d.New.ObjectName)

	// Generate REVOKE for removed privileges
	if len(toRevoke) > 0 {
		sort.Strings(toRevoke)
		statements = append(statements, fmt.Sprintf("REVOKE %s ON %s FROM %s;",
			strings.Join(toRevoke, ", "), objectRef, grantee))
	}

	// Generate GRANT for added privileges
	if len(toGrant) > 0 {
		sort.Strings(toGrant)
		sql := fmt.Sprintf("GRANT %s ON %s TO %s", strings.Join(toGrant, ", "), objectRef, grantee)
		if d.New.WithGrantOption {
			sql += " WITH GRANT OPTION"
		}
		statements = append(statements, sql+";")
	}

	// Handle WITH GRANT OPTION changes for unchanged privileges
	if d.Old.WithGrantOption != d.New.WithGrantOption {
		// Find unchanged privileges (in both old and new)
		var unchanged []string
		for p := range oldPrivSet {
			if newPrivSet[p] {
				unchanged = append(unchanged, p)
			}
		}

		if len(unchanged) > 0 {
			sort.Strings(unchanged)
			unchangedStr := strings.Join(unchanged, ", ")

			if d.Old.WithGrantOption && !d.New.WithGrantOption {
				// Revoke grant option only (keep the privilege)
				statements = append(statements, fmt.Sprintf("REVOKE GRANT OPTION FOR %s ON %s FROM %s;",
					unchangedStr, objectRef, grantee))
			} else if !d.Old.WithGrantOption && d.New.WithGrantOption {
				// Add grant option (re-grant with grant option)
				statements = append(statements, fmt.Sprintf("GRANT %s ON %s TO %s WITH GRANT OPTION;",
					unchangedStr, objectRef, grantee))
			}
		}
	}

	return statements
}

// generateRevokeDefaultPrivilegesSQL generates REVOKE statements for revoking default PUBLIC grants
func generateRevokeDefaultPrivilegesSQL(revoked []*ir.RevokedDefaultPrivilege, targetSchema string, collector *diffCollector) {
	for _, r := range revoked {
		sql := generateRevokeDefaultPublicSQL(r)

		context := &diffContext{
			Type:                DiffTypeRevokedDefaultPrivilege,
			Operation:           DiffOperationCreate, // Creating a revoke
			Path:                fmt.Sprintf("revoked_default.%s.%s", r.ObjectType, r.ObjectName),
			Source:              r,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateRestoreDefaultPrivilegesSQL generates GRANT statements to restore default PUBLIC grants
func generateRestoreDefaultPrivilegesSQL(revoked []*ir.RevokedDefaultPrivilege, targetSchema string, collector *diffCollector) {
	for _, r := range revoked {
		sql := generateGrantDefaultPublicSQL(r)

		context := &diffContext{
			Type:                DiffTypeRevokedDefaultPrivilege,
			Operation:           DiffOperationDrop, // Dropping a revoke = restoring default
			Path:                fmt.Sprintf("revoked_default.%s.%s", r.ObjectType, r.ObjectName),
			Source:              r,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateRevokeDefaultPublicSQL generates REVOKE ... FROM PUBLIC statement
func generateRevokeDefaultPublicSQL(r *ir.RevokedDefaultPrivilege) string {
	sortedPrivs := make([]string, len(r.Privileges))
	copy(sortedPrivs, r.Privileges)
	sort.Strings(sortedPrivs)

	privStr := strings.Join(sortedPrivs, ", ")
	objectRef := formatObjectReference(r.ObjectType, r.ObjectName)

	return fmt.Sprintf("REVOKE %s ON %s FROM PUBLIC;", privStr, objectRef)
}

// generateGrantDefaultPublicSQL generates GRANT ... TO PUBLIC statement (restore default)
func generateGrantDefaultPublicSQL(r *ir.RevokedDefaultPrivilege) string {
	sortedPrivs := make([]string, len(r.Privileges))
	copy(sortedPrivs, r.Privileges)
	sort.Strings(sortedPrivs)

	privStr := strings.Join(sortedPrivs, ", ")
	objectRef := formatObjectReference(r.ObjectType, r.ObjectName)

	return fmt.Sprintf("GRANT %s ON %s TO PUBLIC;", privStr, objectRef)
}

// formatGrantee formats the grantee for use in GRANT/REVOKE statements
func formatGrantee(grantee string) string {
	if grantee == "" || grantee == "PUBLIC" {
		return "PUBLIC"
	}
	return ir.QuoteIdentifier(grantee)
}

// formatObjectReference formats the object reference for GRANT/REVOKE statements
func formatObjectReference(objType ir.PrivilegeObjectType, objName string) string {
	switch objType {
	case ir.PrivilegeObjectTypeTable:
		return "TABLE " + ir.QuoteIdentifier(objName)
	case ir.PrivilegeObjectTypeView:
		return "TABLE " + ir.QuoteIdentifier(objName) // Views use TABLE keyword in GRANT
	case ir.PrivilegeObjectTypeSequence:
		return "SEQUENCE " + ir.QuoteIdentifier(objName)
	case ir.PrivilegeObjectTypeFunction:
		return "FUNCTION " + objName // Function signature already includes parentheses
	case ir.PrivilegeObjectTypeProcedure:
		return "PROCEDURE " + objName // Procedure signature already includes parentheses
	case ir.PrivilegeObjectTypeType:
		return "TYPE " + ir.QuoteIdentifier(objName)
	default:
		return ir.QuoteIdentifier(objName)
	}
}

// privilegesEqual checks if two privileges are structurally equal
func privilegesEqual(old, new *ir.Privilege) bool {
	if old.ObjectType != new.ObjectType {
		return false
	}
	if old.ObjectName != new.ObjectName {
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

// GetObjectName returns a unique identifier for the privilege diff
func (d *privilegeDiff) GetObjectName() string {
	return d.New.GetObjectKey()
}

// isPrivilegeCoveredByDefaultPrivileges checks if an explicit privilege is covered
// by default privileges in the desired state. This is used to avoid generating
// spurious REVOKE statements for privileges that are auto-granted via default privileges.
// See https://github.com/pgschema/pgschema/issues/250
func isPrivilegeCoveredByDefaultPrivileges(p *ir.Privilege, defaultPrivileges []*ir.DefaultPrivilege) bool {
	for _, dp := range defaultPrivileges {
		// Match object types (TABLE -> TABLES, SEQUENCE -> SEQUENCES, etc.)
		if !privilegeObjectTypeMatchesDefault(p.ObjectType, dp.ObjectType) {
			continue
		}

		// Match grantee
		if p.Grantee != dp.Grantee {
			continue
		}

		// Match grant option
		if p.WithGrantOption != dp.WithGrantOption {
			continue
		}

		// Check if all privilege types are covered by the default privilege
		if privilegesCoveredBy(p.Privileges, dp.Privileges) {
			return true
		}
	}
	return false
}

// privilegeObjectTypeMatchesDefault checks if a privilege object type matches
// a default privilege object type (e.g., TABLE matches TABLES)
func privilegeObjectTypeMatchesDefault(privType ir.PrivilegeObjectType, defaultType ir.DefaultPrivilegeObjectType) bool {
	switch privType {
	case ir.PrivilegeObjectTypeTable:
		return defaultType == ir.DefaultPrivilegeObjectTypeTables
	case ir.PrivilegeObjectTypeSequence:
		return defaultType == ir.DefaultPrivilegeObjectTypeSequences
	case ir.PrivilegeObjectTypeFunction:
		return defaultType == ir.DefaultPrivilegeObjectTypeFunctions
	case ir.PrivilegeObjectTypeProcedure:
		return defaultType == ir.DefaultPrivilegeObjectTypeFunctions // Procedures use FUNCTIONS default
	case ir.PrivilegeObjectTypeType:
		return defaultType == ir.DefaultPrivilegeObjectTypeTypes
	default:
		return false
	}
}

// privilegesCoveredBy checks if all privileges in 'privs' are covered by 'coveringPrivs'
func privilegesCoveredBy(privs, coveringPrivs []string) bool {
	coveringSet := make(map[string]bool)
	for _, p := range coveringPrivs {
		coveringSet[p] = true
	}
	for _, p := range privs {
		if !coveringSet[p] {
			return false
		}
	}
	return true
}

// computeRevokedDefaultGrants finds privileges that would be auto-granted by default privileges
// on new objects, but should be explicitly revoked because the user didn't include them in the new state.
// See https://github.com/pgschema/pgschema/issues/253
func computeRevokedDefaultGrants(addedTables []*ir.Table, newPrivs map[string]*ir.Privilege, defaultPrivileges []*ir.DefaultPrivilege) []*ir.Privilege {
	var revokedPrivs []*ir.Privilege

	// For each new table, check which default privileges would auto-grant
	for _, table := range addedTables {
		for _, dp := range defaultPrivileges {
			// Only process default privileges for TABLES
			if dp.ObjectType != ir.DefaultPrivilegeObjectTypeTables {
				continue
			}

			// Check if the new state has an explicit privilege for this table+grantee
			// If not, the user intentionally revoked the default grant
			privilegeKey := string(ir.PrivilegeObjectTypeTable) + ":" + table.Name + ":" + dp.Grantee
			hasExplicitGrant := false
			for key := range newPrivs {
				// Match by object key prefix (without grant option suffix)
				if len(key) >= len(privilegeKey) && key[:len(privilegeKey)] == privilegeKey {
					hasExplicitGrant = true
					break
				}
			}

			if !hasExplicitGrant {
				// Create a synthetic privilege to revoke the default grant
				revokedPrivs = append(revokedPrivs, &ir.Privilege{
					ObjectType:      ir.PrivilegeObjectTypeTable,
					ObjectName:      table.Name,
					Grantee:         dp.Grantee,
					Privileges:      dp.Privileges,
					WithGrantOption: dp.WithGrantOption,
				})
			}
		}
	}

	return revokedPrivs
}
