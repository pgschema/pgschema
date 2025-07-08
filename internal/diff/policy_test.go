package diff

import (
	"testing"

	"github.com/pgschema/pgschema/internal/ir"
)

func TestPoliciesEqual(t *testing.T) {
	tests := []struct {
		name     string
		old      *ir.RLSPolicy
		new      *ir.RLSPolicy
		expected bool
	}{
		{
			name: "identical policies",
			old: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
				WithCheck:  "user_id = current_user_id()",
			},
			new: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
				WithCheck:  "user_id = current_user_id()",
			},
			expected: true,
		},
		{
			name: "different names",
			old: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			new: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_policy",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			expected: false,
		},
		{
			name: "different commands",
			old: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			new: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandSelect,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			expected: false,
		},
		{
			name: "different permissive",
			old: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			new: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: false,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			expected: false,
		},
		{
			name: "different roles",
			old: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			new: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"user_role"},
				Using:      "user_id = current_user_id()",
			},
			expected: false,
		},
		{
			name: "different using clause",
			old: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			new: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "tenant_id = current_tenant_id()",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := policiesEqual(tt.old, tt.new)
			if result != tt.expected {
				t.Errorf("policiesEqual() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNeedsRecreate(t *testing.T) {
	tests := []struct {
		name     string
		old      *ir.RLSPolicy
		new      *ir.RLSPolicy
		expected bool
	}{
		{
			name: "name change requires recreation",
			old: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			new: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_policy",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			expected: true,
		},
		{
			name: "command change requires recreation",
			old: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			new: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandSelect,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			expected: true,
		},
		{
			name: "permissive change requires recreation",
			old: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			new: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: false,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			expected: true,
		},
		{
			name: "role change can be altered",
			old: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			new: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"user_role"},
				Using:      "user_id = current_user_id()",
			},
			expected: false,
		},
		{
			name: "using clause change can be altered",
			old: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
			},
			new: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "tenant_id = current_tenant_id()",
			},
			expected: false,
		},
		{
			name: "with check change can be altered",
			old: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
				WithCheck:  "user_id = current_user_id()",
			},
			new: &ir.RLSPolicy{
				Schema:     "public",
				Table:      "users",
				Name:       "user_isolation",
				Command:    ir.PolicyCommandAll,
				Permissive: true,
				Roles:      []string{"PUBLIC"},
				Using:      "user_id = current_user_id()",
				WithCheck:  "tenant_id = current_tenant_id()",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsRecreate(tt.old, tt.new)
			if result != tt.expected {
				t.Errorf("needsRecreate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateCreatePolicySQL(t *testing.T) {
	tests := []struct {
		name     string
		policies []*ir.RLSPolicy
		expected []string
	}{
		{
			name: "simple policy",
			policies: []*ir.RLSPolicy{
				{
					Schema:     "public",
					Table:      "users",
					Name:       "user_isolation",
					Command:    ir.PolicyCommandAll,
					Permissive: true,
					Roles:      []string{"PUBLIC"},
					Using:      "user_id = current_user_id()",
				},
			},
			expected: []string{
				"CREATE POLICY user_isolation ON users TO PUBLIC USING (user_id = current_user_id());",
			},
		},
		{
			name: "restrictive policy with specific command",
			policies: []*ir.RLSPolicy{
				{
					Schema:     "public",
					Table:      "users",
					Name:       "admin_only",
					Command:    ir.PolicyCommandSelect,
					Permissive: false,
					Roles:      []string{"admin"},
					Using:      "role = 'admin'",
				},
			},
			expected: []string{
				"CREATE POLICY admin_only ON users AS RESTRICTIVE FOR SELECT TO admin USING (role = 'admin');",
			},
		},
		{
			name: "policy with WITH CHECK clause",
			policies: []*ir.RLSPolicy{
				{
					Schema:     "public",
					Table:      "audit",
					Name:       "audit_insert",
					Command:    ir.PolicyCommandInsert,
					Permissive: true,
					Roles:      []string{"PUBLIC"},
					WithCheck:  "true",
				},
			},
			expected: []string{
				"CREATE POLICY audit_insert ON audit FOR INSERT TO PUBLIC WITH CHECK (true);",
			},
		},
		{
			name: "policy with multiple roles",
			policies: []*ir.RLSPolicy{
				{
					Schema:     "public",
					Table:      "users",
					Name:       "multi_role",
					Command:    ir.PolicyCommandAll,
					Permissive: true,
					Roles:      []string{"user_role", "admin_role"},
					Using:      "tenant_id = current_tenant_id()",
				},
			},
			expected: []string{
				"CREATE POLICY multi_role ON users TO user_role, admin_role USING (tenant_id = current_tenant_id());",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateCreatePolicySQL(tt.policies)
			if len(result) != len(tt.expected) {
				t.Errorf("GenerateCreatePolicySQL() returned %d statements, expected %d", len(result), len(tt.expected))
				return
			}
			for i, stmt := range result {
				if stmt != tt.expected[i] {
					t.Errorf("GenerateCreatePolicySQL() statement %d = %q, want %q", i, stmt, tt.expected[i])
				}
			}
		})
	}
}

func TestGenerateAlterPolicySQL(t *testing.T) {
	tests := []struct {
		name     string
		diffs    []*PolicyDiff
		expected []string
	}{
		{
			name: "alter roles",
			diffs: []*PolicyDiff{
				{
					Old: &ir.RLSPolicy{
						Schema:     "public",
						Table:      "users",
						Name:       "user_isolation",
						Command:    ir.PolicyCommandAll,
						Permissive: true,
						Roles:      []string{"PUBLIC"},
						Using:      "user_id = current_user_id()",
					},
					New: &ir.RLSPolicy{
						Schema:     "public",
						Table:      "users",
						Name:       "user_isolation",
						Command:    ir.PolicyCommandAll,
						Permissive: true,
						Roles:      []string{"user_role"},
						Using:      "user_id = current_user_id()",
					},
				},
			},
			expected: []string{
				"ALTER POLICY user_isolation ON users TO user_role;",
			},
		},
		{
			name: "alter using clause",
			diffs: []*PolicyDiff{
				{
					Old: &ir.RLSPolicy{
						Schema:     "public",
						Table:      "users",
						Name:       "user_isolation",
						Command:    ir.PolicyCommandAll,
						Permissive: true,
						Roles:      []string{"PUBLIC"},
						Using:      "user_id = current_user_id()",
					},
					New: &ir.RLSPolicy{
						Schema:     "public",
						Table:      "users",
						Name:       "user_isolation",
						Command:    ir.PolicyCommandAll,
						Permissive: true,
						Roles:      []string{"PUBLIC"},
						Using:      "tenant_id = current_tenant_id()",
					},
				},
			},
			expected: []string{
				"ALTER POLICY user_isolation ON users USING (tenant_id = current_tenant_id());",
			},
		},
		{
			name: "recreate for name change",
			diffs: []*PolicyDiff{
				{
					Old: &ir.RLSPolicy{
						Schema:     "public",
						Table:      "users",
						Name:       "user_isolation",
						Command:    ir.PolicyCommandAll,
						Permissive: true,
						Roles:      []string{"PUBLIC"},
						Using:      "user_id = current_user_id()",
					},
					New: &ir.RLSPolicy{
						Schema:     "public",
						Table:      "users",
						Name:       "user_policy",
						Command:    ir.PolicyCommandAll,
						Permissive: true,
						Roles:      []string{"PUBLIC"},
						Using:      "user_id = current_user_id()",
					},
				},
			},
			expected: []string{
				"DROP POLICY IF EXISTS user_isolation ON users;",
				"CREATE POLICY user_policy ON users TO PUBLIC USING (user_id = current_user_id());",
			},
		},
		{
			name: "recreate for command change",
			diffs: []*PolicyDiff{
				{
					Old: &ir.RLSPolicy{
						Schema:     "public",
						Table:      "users",
						Name:       "user_isolation",
						Command:    ir.PolicyCommandAll,
						Permissive: true,
						Roles:      []string{"PUBLIC"},
						Using:      "user_id = current_user_id()",
					},
					New: &ir.RLSPolicy{
						Schema:     "public",
						Table:      "users",
						Name:       "user_isolation",
						Command:    ir.PolicyCommandSelect,
						Permissive: true,
						Roles:      []string{"PUBLIC"},
						Using:      "user_id = current_user_id()",
					},
				},
			},
			expected: []string{
				"DROP POLICY IF EXISTS user_isolation ON users;",
				"CREATE POLICY user_isolation ON users FOR SELECT TO PUBLIC USING (user_id = current_user_id());",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateAlterPolicySQL(tt.diffs)
			if len(result) != len(tt.expected) {
				t.Errorf("GenerateAlterPolicySQL() returned %d statements, expected %d", len(result), len(tt.expected))
				return
			}
			for i, stmt := range result {
				if stmt != tt.expected[i] {
					t.Errorf("GenerateAlterPolicySQL() statement %d = %q, want %q", i, stmt, tt.expected[i])
				}
			}
		})
	}
}

func TestGenerateRLSChangeSQL(t *testing.T) {
	tests := []struct {
		name     string
		changes  []*RLSChange
		expected []string
	}{
		{
			name: "enable RLS",
			changes: []*RLSChange{
				{
					Table: &ir.Table{
						Schema: "public",
						Name:   "users",
					},
					Enabled: true,
				},
			},
			expected: []string{
				"ALTER TABLE users ENABLE ROW LEVEL SECURITY;",
			},
		},
		{
			name: "disable RLS",
			changes: []*RLSChange{
				{
					Table: &ir.Table{
						Schema: "public",
						Name:   "users",
					},
					Enabled: false,
				},
			},
			expected: []string{
				"ALTER TABLE users DISABLE ROW LEVEL SECURITY;",
			},
		},
		{
			name: "multiple changes",
			changes: []*RLSChange{
				{
					Table: &ir.Table{
						Schema: "public",
						Name:   "audit",
					},
					Enabled: true,
				},
				{
					Table: &ir.Table{
						Schema: "public",
						Name:   "users",
					},
					Enabled: false,
				},
			},
			expected: []string{
				"ALTER TABLE audit ENABLE ROW LEVEL SECURITY;",
				"ALTER TABLE users DISABLE ROW LEVEL SECURITY;",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateRLSChangeSQL(tt.changes)
			if len(result) != len(tt.expected) {
				t.Errorf("GenerateRLSChangeSQL() returned %d statements, expected %d", len(result), len(tt.expected))
				return
			}
			for i, stmt := range result {
				if stmt != tt.expected[i] {
					t.Errorf("GenerateRLSChangeSQL() statement %d = %q, want %q", i, stmt, tt.expected[i])
				}
			}
		})
	}
}
