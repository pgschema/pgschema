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
	mode  OperationMode
}

// newDiffCollector creates a new diffCollector for plan mode (backward compatibility)
func newDiffCollector() *diffCollector {
	return newDiffCollectorWithMode(PlanMode)
}

// newDiffCollectorWithMode creates a new diffCollector with the specified operation mode
func newDiffCollectorWithMode(mode OperationMode) *diffCollector {
	return &diffCollector{
		diffs: []Diff{},
		mode:  mode,
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

// collectMultipleStatements collects multiple related SQL statements as a single Diff
func (c *diffCollector) collectMultipleStatements(context *diffContext, statements []SQLStatement) {
	if context != nil && len(statements) > 0 {
		step := Diff{
			Statements: statements,
			Type:       context.Type,
			Operation:  context.Operation,
			Path:       context.Path,
			Source:     context.Source,
		}
		c.diffs = append(c.diffs, step)
	}
}
