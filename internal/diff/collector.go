package diff


// diffContext provides context about the SQL statement being generated
type diffContext struct {
	Type                DiffType      // e.g., DiffTypeTable, DiffTypeView, DiffTypeFunction
	Operation           DiffOperation // e.g., DiffOperationCreate, DiffOperationAlter, DiffOperationDrop
	Path                string        // e.g., "schema.table" or "schema.table.column"
	Source              any           // The ddlDiff element that generated this SQL
	CanRunInTransaction bool          // Whether this SQL can run in a transaction
}

// diffCollector collects SQL statements with their context information
type diffCollector struct {
	diffs []Diff
}

// newDiffCollector creates a new diffCollector
func newDiffCollector() *diffCollector {
	return &diffCollector{
		diffs: []Diff{},
	}
}

// collect collects a single SQL statement with its context information
func (c *diffCollector) collect(context *diffContext, stmt string) {
	if context != nil {
		step := Diff{
			Statements: []SQLStatement{{
				SQL:                 stmt,
				CanRunInTransaction: context.CanRunInTransaction,
			}},
			Type:      context.Type,
			Operation: context.Operation,
			Path:      context.Path,
			Source:    context.Source,
		}
		c.diffs = append(c.diffs, step)
	}
}

// collectWithRewrite collects a statement with optional rewrite for online operations
func (c *diffCollector) collectWithRewrite(context *diffContext, stmt string, rewrite *DiffRewrite) {
	if context != nil {
		step := Diff{
			Statements: []SQLStatement{{
				SQL:                 stmt,
				CanRunInTransaction: context.CanRunInTransaction,
			}},
			Type:      context.Type,
			Operation: context.Operation,
			Path:      context.Path,
			Source:    context.Source,
			Rewrite:   rewrite,
		}
		c.diffs = append(c.diffs, step)
	}
}

// collectStatement collects a pre-built SQLStatement with its context information
func (c *diffCollector) collectStatement(context *diffContext, statement SQLStatement) {
	if context != nil {
		step := Diff{
			Statements: []SQLStatement{statement},
			Type:       context.Type,
			Operation:  context.Operation,
			Path:       context.Path,
			Source:     context.Source,
		}
		c.diffs = append(c.diffs, step)
	}
}

// collectMultipleStatements collects multiple SQL statements in a single diff with optional rewrite
func (c *diffCollector) collectMultipleStatements(context *diffContext, statements []SQLStatement, rewrite *DiffRewrite) {
	if context != nil && len(statements) > 0 {
		step := Diff{
			Statements: statements,
			Type:       context.Type,
			Operation:  context.Operation,
			Path:       context.Path,
			Source:     context.Source,
			Rewrite:    rewrite,
		}
		c.diffs = append(c.diffs, step)
	}
}

