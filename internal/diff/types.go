package diff

import (
	"github.com/pgschema/pgschema/internal/ir"
)

// DDLDiff represents the difference between two DDL states
type DDLDiff struct {
	AddedSchemas      []*ir.Schema
	DroppedSchemas    []*ir.Schema
	ModifiedSchemas   []*SchemaDiff
	AddedTables       []*ir.Table
	DroppedTables     []*ir.Table
	ModifiedTables    []*TableDiff
	AddedViews        []*ir.View
	DroppedViews      []*ir.View
	ModifiedViews     []*ViewDiff
	AddedExtensions   []*ir.Extension
	DroppedExtensions []*ir.Extension
	AddedFunctions    []*ir.Function
	DroppedFunctions  []*ir.Function
	ModifiedFunctions []*FunctionDiff
	AddedIndexes      []*ir.Index
	DroppedIndexes    []*ir.Index
	AddedTypes        []*ir.Type
	DroppedTypes      []*ir.Type
	ModifiedTypes     []*TypeDiff
	AddedTriggers     []*ir.Trigger
	DroppedTriggers   []*ir.Trigger
	ModifiedTriggers  []*TriggerDiff
}

// SchemaDiff represents changes to a schema
type SchemaDiff struct {
	Old *ir.Schema
	New *ir.Schema
}

// FunctionDiff represents changes to a function
type FunctionDiff struct {
	Old *ir.Function
	New *ir.Function
}

// TypeDiff represents changes to a type
type TypeDiff struct {
	Old *ir.Type
	New *ir.Type
}

// TriggerDiff represents changes to a trigger
type TriggerDiff struct {
	Old *ir.Trigger
	New *ir.Trigger
}

// ViewDiff represents changes to a view
type ViewDiff struct {
	Old *ir.View
	New *ir.View
}

// TableDiff represents changes to a table
type TableDiff struct {
	Table              *ir.Table
	AddedColumns       []*ir.Column
	DroppedColumns     []*ir.Column
	ModifiedColumns    []*ColumnDiff
	AddedConstraints   []*ir.Constraint
	DroppedConstraints []*ir.Constraint
}

// ColumnDiff represents changes to a column
type ColumnDiff struct {
	Old *ir.Column
	New *ir.Column
}