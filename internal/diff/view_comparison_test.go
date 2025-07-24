package diff

import (
	"testing"

	"github.com/pgschema/pgschema/internal/ir"
)

func TestViewSemanticComparison(t *testing.T) {
	tests := []struct {
		name        string
		definition1 string
		definition2 string
		expectEqual bool
	}{
		{
			name: "identical views",
			definition1: ` SELECT
    emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM dept_emp
  GROUP BY emp_no`,
			definition2: ` SELECT
    emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM dept_emp
  GROUP BY emp_no`,
			expectEqual: true,
		},
		{
			name: "formatting differences - semicolon and line breaks",
			definition1: ` SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM dept_emp
  GROUP BY emp_no;`,
			definition2: ` SELECT
    emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM dept_emp
  GROUP BY emp_no`,
			expectEqual: true,
		},
		{
			name: "complex view with joins - formatting differences",
			definition1: ` SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM dept_emp d
     JOIN dept_emp_latest_date l ON d.emp_no = l.emp_no AND d.from_date = l.from_date AND l.to_date = d.to_date;`,
			definition2: ` SELECT
    l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM dept_emp d
     JOIN dept_emp_latest_date l ON d.emp_no = l.emp_no AND d.from_date = l.from_date AND l.to_date = d.to_date`,
			expectEqual: true,
		},
		{
			name: "different column order",
			definition1: ` SELECT emp_no, name FROM users`,
			definition2: ` SELECT name, emp_no FROM users`,
			expectEqual: false,
		},
		{
			name: "different function",
			definition1: ` SELECT emp_no, max(from_date) FROM dept_emp GROUP BY emp_no`,
			definition2: ` SELECT emp_no, min(from_date) FROM dept_emp GROUP BY emp_no`,
			expectEqual: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the views as IR objects
			view1 := &ir.View{
				Schema:     "public",
				Name:       "test_view",
				Definition: tt.definition1,
			}
			view2 := &ir.View{
				Schema:     "public",
				Name:       "test_view",
				Definition: tt.definition2,
			}

			result := viewsEqual(view1, view2)
			if result != tt.expectEqual {
				t.Errorf("viewsEqual() = %v, expected %v", result, tt.expectEqual)
				t.Logf("Definition 1:\n%s", tt.definition1)
				t.Logf("Definition 2:\n%s", tt.definition2)
			}

			// Also test the semantic comparison function directly
			semanticResult := compareViewDefinitionsSemanticially(tt.definition1, tt.definition2)
			if semanticResult != tt.expectEqual {
				t.Errorf("compareViewDefinitionsSemanticially() = %v, expected %v", semanticResult, tt.expectEqual)
			}
		})
	}
}

func TestFunctionCallComparison(t *testing.T) {
	// Specific test for the issue we fixed - function calls with different location metadata
	tests := []struct {
		name        string
		sql1        string
		sql2        string
		expectEqual bool
	}{
		{
			name: "max function with different formatting",
			sql1: " SELECT emp_no, max(from_date) AS from_date FROM dept_emp GROUP BY emp_no",
			sql2: " SELECT\n    emp_no,\n    max(from_date) AS from_date\n   FROM dept_emp\n  GROUP BY emp_no",
			expectEqual: true,
		},
		{
			name: "count function with different formatting",
			sql1: " SELECT count(*) FROM users",
			sql2: " SELECT\n    count(*)\n   FROM users",
			expectEqual: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareViewDefinitionsSemanticially(tt.sql1, tt.sql2)
			if result != tt.expectEqual {
				t.Errorf("compareViewDefinitionsSemanticially() = %v, expected %v", result, tt.expectEqual)
				t.Logf("SQL 1:\n%s", tt.sql1)
				t.Logf("SQL 2:\n%s", tt.sql2)
			}
		})
	}
}