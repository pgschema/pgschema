package diff

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/ir"
)

// DiffType represents the type of database object being changed
type DiffType int

const (
	DiffTypeTable DiffType = iota
	DiffTypeTableColumn
	DiffTypeTableIndex
	DiffTypeTableTrigger
	DiffTypeTablePolicy
	DiffTypeTableRLS
	DiffTypeTableConstraint
	DiffTypeTableComment
	DiffTypeTableColumnComment
	DiffTypeTableIndexComment
	DiffTypeView
	DiffTypeViewComment
	DiffTypeMaterializedView
	DiffTypeMaterializedViewComment
	DiffTypeMaterializedViewIndex
	DiffTypeMaterializedViewIndexComment
	DiffTypeFunction
	DiffTypeProcedure
	DiffTypeSequence
	DiffTypeType
	DiffTypeDomain
	DiffTypeComment
	DiffTypeDefaultPrivilege
	DiffTypePrivilege
	DiffTypeRevokedDefaultPrivilege
)

// String returns the string representation of DiffType
func (d DiffType) String() string {
	switch d {
	case DiffTypeTable:
		return "table"
	case DiffTypeTableColumn:
		return "table.column"
	case DiffTypeTableIndex:
		return "table.index"
	case DiffTypeTableTrigger:
		return "table.trigger"
	case DiffTypeTablePolicy:
		return "table.policy"
	case DiffTypeTableRLS:
		return "table.rls"
	case DiffTypeTableConstraint:
		return "table.constraint"
	case DiffTypeTableComment:
		return "table.comment"
	case DiffTypeTableColumnComment:
		return "table.column.comment"
	case DiffTypeTableIndexComment:
		return "table.index.comment"
	case DiffTypeView:
		return "view"
	case DiffTypeViewComment:
		return "view.comment"
	case DiffTypeMaterializedView:
		return "materialized_view"
	case DiffTypeMaterializedViewComment:
		return "materialized_view.comment"
	case DiffTypeMaterializedViewIndex:
		return "materialized_view.index"
	case DiffTypeMaterializedViewIndexComment:
		return "materialized_view.index.comment"
	case DiffTypeFunction:
		return "function"
	case DiffTypeProcedure:
		return "procedure"
	case DiffTypeSequence:
		return "sequence"
	case DiffTypeType:
		return "type"
	case DiffTypeDomain:
		return "domain"
	case DiffTypeComment:
		return "comment"
	case DiffTypeDefaultPrivilege:
		return "default_privilege"
	case DiffTypePrivilege:
		return "privilege"
	case DiffTypeRevokedDefaultPrivilege:
		return "revoked_default_privilege"
	default:
		return "unknown"
	}
}

// MarshalJSON marshals DiffType to JSON as a string
func (d DiffType) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON unmarshals DiffType from JSON string
func (d *DiffType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case "table":
		*d = DiffTypeTable
	case "table.column":
		*d = DiffTypeTableColumn
	case "table.index":
		*d = DiffTypeTableIndex
	case "table.trigger":
		*d = DiffTypeTableTrigger
	case "table.policy":
		*d = DiffTypeTablePolicy
	case "table.rls":
		*d = DiffTypeTableRLS
	case "table.constraint":
		*d = DiffTypeTableConstraint
	case "table.comment":
		*d = DiffTypeTableComment
	case "table.column.comment":
		*d = DiffTypeTableColumnComment
	case "table.index.comment":
		*d = DiffTypeTableIndexComment
	case "view":
		*d = DiffTypeView
	case "view.comment":
		*d = DiffTypeViewComment
	case "materialized_view":
		*d = DiffTypeMaterializedView
	case "materialized_view.comment":
		*d = DiffTypeMaterializedViewComment
	case "materialized_view.index":
		*d = DiffTypeMaterializedViewIndex
	case "materialized_view.index.comment":
		*d = DiffTypeMaterializedViewIndexComment
	case "function":
		*d = DiffTypeFunction
	case "procedure":
		*d = DiffTypeProcedure
	case "sequence":
		*d = DiffTypeSequence
	case "type":
		*d = DiffTypeType
	case "domain":
		*d = DiffTypeDomain
	case "comment":
		*d = DiffTypeComment
	case "default_privilege":
		*d = DiffTypeDefaultPrivilege
	case "privilege":
		*d = DiffTypePrivilege
	case "revoked_default_privilege":
		*d = DiffTypeRevokedDefaultPrivilege
	default:
		return fmt.Errorf("unknown diff type: %s", s)
	}
	return nil
}

// DiffOperation represents the operation being performed
type DiffOperation int

const (
	DiffOperationCreate DiffOperation = iota
	DiffOperationAlter
	DiffOperationDrop
	// DiffOperationRecreate indicates a DROP that is part of a DROP+CREATE cycle
	// for dependency handling. The object will be recreated, so this should be
	// counted as a modification in summaries, not a destruction.
	DiffOperationRecreate
)

// String returns the string representation of DiffOperation
func (d DiffOperation) String() string {
	switch d {
	case DiffOperationCreate:
		return "create"
	case DiffOperationAlter:
		return "alter"
	case DiffOperationDrop:
		return "drop"
	case DiffOperationRecreate:
		return "recreate"
	default:
		return "unknown"
	}
}

// MarshalJSON marshals DiffOperation to JSON as a string
func (d DiffOperation) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON unmarshals DiffOperation from JSON string
func (d *DiffOperation) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch s {
	case "create":
		*d = DiffOperationCreate
	case "alter":
		*d = DiffOperationAlter
	case "drop":
		*d = DiffOperationDrop
	case "recreate":
		*d = DiffOperationRecreate
	default:
		return fmt.Errorf("unknown diff operation: %s", s)
	}
	return nil
}

// DiffSource represents all possible source types for a diff
type DiffSource interface {
	GetObjectName() string // Returns the object name (preserves names with dots like "public.idx_users")
}

// SQLStatement represents a single SQL statement with its transaction capability
type SQLStatement struct {
	SQL                 string `json:"sql,omitempty"`
	CanRunInTransaction bool   `json:"can_run_in_transaction"`
}

// Diff represents one or more related SQL statements with their source change
type Diff struct {
	Statements []SQLStatement `json:"statements"`
	Type       DiffType       `json:"type"`
	Operation  DiffOperation  `json:"operation"` // create, alter, drop, replace
	Path       string         `json:"path"`
	Source     DiffSource     `json:"source,omitempty"`
}

type ddlDiff struct {
	addedSchemas       []*ir.Schema
	droppedSchemas     []*ir.Schema
	modifiedSchemas    []*schemaDiff
	addedTables        []*ir.Table
	droppedTables      []*ir.Table
	modifiedTables     []*tableDiff
	addedViews         []*ir.View
	droppedViews       []*ir.View
	modifiedViews      []*viewDiff
	addedFunctions     []*ir.Function
	droppedFunctions   []*ir.Function
	modifiedFunctions  []*functionDiff
	addedProcedures    []*ir.Procedure
	droppedProcedures  []*ir.Procedure
	modifiedProcedures []*procedureDiff
	addedTypes         []*ir.Type
	droppedTypes       []*ir.Type
	modifiedTypes      []*typeDiff
	addedSequences            []*ir.Sequence
	droppedSequences          []*ir.Sequence
	modifiedSequences         []*sequenceDiff
	addedDefaultPrivileges    []*ir.DefaultPrivilege
	droppedDefaultPrivileges  []*ir.DefaultPrivilege
	modifiedDefaultPrivileges []*defaultPrivilegeDiff
	// Explicit object privileges
	addedPrivileges              []*ir.Privilege
	droppedPrivileges            []*ir.Privilege
	modifiedPrivileges           []*privilegeDiff
	addedRevokedDefaultPrivs     []*ir.RevokedDefaultPrivilege
	droppedRevokedDefaultPrivs   []*ir.RevokedDefaultPrivilege
}

// schemaDiff represents changes to a schema
type schemaDiff struct {
	Old *ir.Schema
	New *ir.Schema
}

// functionDiff represents changes to a function
type functionDiff struct {
	Old *ir.Function
	New *ir.Function
}

// procedureDiff represents changes to a procedure
type procedureDiff struct {
	Old *ir.Procedure
	New *ir.Procedure
}

// typeDiff represents changes to a type
type typeDiff struct {
	Old *ir.Type
	New *ir.Type
}

// sequenceDiff represents changes to a sequence
type sequenceDiff struct {
	Old *ir.Sequence
	New *ir.Sequence
}

// defaultPrivilegeDiff represents changes to default privileges
type defaultPrivilegeDiff struct {
	Old *ir.DefaultPrivilege
	New *ir.DefaultPrivilege
}

// privilegeDiff represents changes to explicit object privileges
type privilegeDiff struct {
	Old *ir.Privilege
	New *ir.Privilege
}

// triggerDiff represents changes to a trigger
type triggerDiff struct {
	Old *ir.Trigger
	New *ir.Trigger
}

// viewDiff represents changes to a view
type viewDiff struct {
	Old              *ir.View
	New              *ir.View
	CommentChanged   bool
	OldComment       string
	NewComment       string
	AddedIndexes     []*ir.Index  // For materialized views
	DroppedIndexes   []*ir.Index  // For materialized views
	ModifiedIndexes  []*IndexDiff // For materialized views
	RequiresRecreate bool         // For materialized views with structural changes that require DROP + CREATE
}

// tableDiff represents changes to a table
type tableDiff struct {
	Table               *ir.Table
	AddedColumns        []*ir.Column
	DroppedColumns      []*ir.Column
	ModifiedColumns     []*ColumnDiff
	AddedConstraints    []*ir.Constraint
	DroppedConstraints  []*ir.Constraint
	ModifiedConstraints []*ConstraintDiff
	AddedIndexes        []*ir.Index
	DroppedIndexes      []*ir.Index
	ModifiedIndexes     []*IndexDiff
	AddedTriggers       []*ir.Trigger
	DroppedTriggers     []*ir.Trigger
	ModifiedTriggers    []*triggerDiff
	AddedPolicies       []*ir.RLSPolicy
	DroppedPolicies     []*ir.RLSPolicy
	ModifiedPolicies    []*policyDiff
	RLSChanges          []*rlsChange
	CommentChanged      bool
	OldComment          string
	NewComment          string
}

// ColumnDiff represents changes to a column
type ColumnDiff struct {
	Old *ir.Column
	New *ir.Column
}

// ConstraintDiff represents changes to a constraint
type ConstraintDiff struct {
	Old *ir.Constraint
	New *ir.Constraint
}

// IndexDiff represents changes to an index
type IndexDiff struct {
	Old *ir.Index
	New *ir.Index
}

// policyDiff represents changes to a policy
type policyDiff struct {
	Old *ir.RLSPolicy
	New *ir.RLSPolicy
}

// rlsChange represents enabling/disabling Row Level Security on a table
type rlsChange struct {
	Table   *ir.Table
	Enabled *bool // nil = no change, true = enable, false = disable
	Forced  *bool // nil = no change, true = force, false = no force
}

// GenerateMigration compares two IR schemas and returns the SQL differences
func GenerateMigration(oldIR, newIR *ir.IR, targetSchema string) []Diff {
	diff := &ddlDiff{
		addedSchemas:       []*ir.Schema{},
		droppedSchemas:     []*ir.Schema{},
		modifiedSchemas:    []*schemaDiff{},
		addedTables:        []*ir.Table{},
		droppedTables:      []*ir.Table{},
		modifiedTables:     []*tableDiff{},
		addedViews:         []*ir.View{},
		droppedViews:       []*ir.View{},
		modifiedViews:      []*viewDiff{},
		addedFunctions:     []*ir.Function{},
		droppedFunctions:   []*ir.Function{},
		modifiedFunctions:  []*functionDiff{},
		addedProcedures:    []*ir.Procedure{},
		droppedProcedures:  []*ir.Procedure{},
		modifiedProcedures: []*procedureDiff{},
		addedTypes:         []*ir.Type{},
		droppedTypes:       []*ir.Type{},
		modifiedTypes:      []*typeDiff{},
		addedSequences:            []*ir.Sequence{},
		droppedSequences:          []*ir.Sequence{},
		modifiedSequences:         []*sequenceDiff{},
		addedDefaultPrivileges:    []*ir.DefaultPrivilege{},
		droppedDefaultPrivileges:  []*ir.DefaultPrivilege{},
		modifiedDefaultPrivileges: []*defaultPrivilegeDiff{},
		addedPrivileges:              []*ir.Privilege{},
		droppedPrivileges:            []*ir.Privilege{},
		modifiedPrivileges:           []*privilegeDiff{},
		addedRevokedDefaultPrivs:     []*ir.RevokedDefaultPrivilege{},
		droppedRevokedDefaultPrivs:   []*ir.RevokedDefaultPrivilege{},
	}

	// Compare schemas first in deterministic order
	schemaNames := sortedKeys(newIR.Schemas)
	for _, name := range schemaNames {
		newDBSchema := newIR.Schemas[name]
		// Skip the public schema as it exists by default
		if name == "public" {
			continue
		}

		if oldDBSchema, exists := oldIR.Schemas[name]; exists {
			// Check if schema has changed (owner)
			if oldDBSchema.Owner != newDBSchema.Owner {
				diff.modifiedSchemas = append(diff.modifiedSchemas, &schemaDiff{
					Old: oldDBSchema,
					New: newDBSchema,
				})
			}
		} else {
			// Schema was added
			diff.addedSchemas = append(diff.addedSchemas, newDBSchema)
		}
	}

	// Find dropped schemas in deterministic order
	oldSchemaNames := sortedKeys(oldIR.Schemas)
	for _, name := range oldSchemaNames {
		oldDBSchema := oldIR.Schemas[name]
		// Skip the public schema as it exists by default
		if name == "public" {
			continue
		}

		if _, exists := newIR.Schemas[name]; !exists {
			diff.droppedSchemas = append(diff.droppedSchemas, oldDBSchema)
		}
	}

	// Build maps for efficient lookup
	oldTables := make(map[string]*ir.Table)
	newTables := make(map[string]*ir.Table)

	// Extract tables from all schemas in oldIR
	for _, dbSchema := range oldIR.Schemas {
		for _, table := range dbSchema.Tables {
			key := table.Schema + "." + table.Name
			oldTables[key] = table
		}
	}

	// Extract tables from all schemas in newIR
	for _, dbSchema := range newIR.Schemas {
		for _, table := range dbSchema.Tables {
			key := table.Schema + "." + table.Name
			newTables[key] = table
		}
	}

	// Find added tables
	for key, table := range newTables {
		if _, exists := oldTables[key]; !exists {
			if table.IsExternal {
				// External table is referenced but doesn't exist in current state
				// Treat it as a "modification" to process triggers, but create an empty old table
				emptyOldTable := &ir.Table{
					Schema:      table.Schema,
					Name:        table.Name,
					IsExternal:  true,
					Triggers:    make(map[string]*ir.Trigger),
					Columns:     []*ir.Column{},
					Constraints: make(map[string]*ir.Constraint),
					Indexes:     make(map[string]*ir.Index),
					Policies:    make(map[string]*ir.RLSPolicy),
				}
				if tableDiff := diffExternalTable(emptyOldTable, table); tableDiff != nil {
					diff.modifiedTables = append(diff.modifiedTables, tableDiff)
				}
			} else {
				diff.addedTables = append(diff.addedTables, table)
			}
		}
	}

	// Find dropped tables
	for key, table := range oldTables {
		if _, exists := newTables[key]; !exists {
			// Skip external tables - they are not managed by pgschema
			if !table.IsExternal {
				diff.droppedTables = append(diff.droppedTables, table)
			}
		}
	}

	// Find modified tables
	for key, newTable := range newTables {
		if oldTable, exists := oldTables[key]; exists {
			// Skip table structure changes for external tables, but still process triggers
			if newTable.IsExternal || oldTable.IsExternal {
				// For external tables, only diff triggers (not table structure)
				if tableDiff := diffExternalTable(oldTable, newTable); tableDiff != nil {
					diff.modifiedTables = append(diff.modifiedTables, tableDiff)
				}
			} else {
				if tableDiff := diffTables(oldTable, newTable, targetSchema); tableDiff != nil {
					diff.modifiedTables = append(diff.modifiedTables, tableDiff)
				}
			}
		}
	}

	// Compare functions across all schemas
	oldFunctions := make(map[string]*ir.Function)
	newFunctions := make(map[string]*ir.Function)

	// Extract functions from all schemas in oldIR in deterministic order
	for _, dbSchema := range oldIR.Schemas {
		funcNames := sortedKeys(dbSchema.Functions)
		for _, funcName := range funcNames {
			function := dbSchema.Functions[funcName]
			// funcName already contains signature as name(arguments) from inspector
			key := function.Schema + "." + funcName
			oldFunctions[key] = function
		}
	}

	// Extract functions from all schemas in newIR in deterministic order
	for _, dbSchema := range newIR.Schemas {
		funcNames := sortedKeys(dbSchema.Functions)
		for _, funcName := range funcNames {
			function := dbSchema.Functions[funcName]
			// funcName already contains signature as name(arguments) from inspector
			key := function.Schema + "." + funcName
			newFunctions[key] = function
		}
	}

	// Find added functions in deterministic order
	functionKeys := sortedKeys(newFunctions)
	for _, key := range functionKeys {
		function := newFunctions[key]
		if _, exists := oldFunctions[key]; !exists {
			diff.addedFunctions = append(diff.addedFunctions, function)
		}
	}

	// Find dropped functions in deterministic order
	oldFunctionKeys := sortedKeys(oldFunctions)
	for _, key := range oldFunctionKeys {
		function := oldFunctions[key]
		if _, exists := newFunctions[key]; !exists {
			diff.droppedFunctions = append(diff.droppedFunctions, function)
		}
	}

	// Find modified functions in deterministic order
	for _, key := range functionKeys {
		newFunction := newFunctions[key]
		if oldFunction, exists := oldFunctions[key]; exists {
			if !functionsEqual(oldFunction, newFunction) {
				diff.modifiedFunctions = append(diff.modifiedFunctions, &functionDiff{
					Old: oldFunction,
					New: newFunction,
				})
			}
		}
	}

	// Compare procedures across all schemas
	oldProcedures := make(map[string]*ir.Procedure)
	newProcedures := make(map[string]*ir.Procedure)

	// Extract procedures from all schemas in oldIR in deterministic order
	for _, dbSchema := range oldIR.Schemas {
		procNames := sortedKeys(dbSchema.Procedures)
		for _, procName := range procNames {
			procedure := dbSchema.Procedures[procName]
			// procName already contains signature as name(arguments) from inspector
			key := procedure.Schema + "." + procName
			oldProcedures[key] = procedure
		}
	}

	// Extract procedures from all schemas in newIR in deterministic order
	for _, dbSchema := range newIR.Schemas {
		procNames := sortedKeys(dbSchema.Procedures)
		for _, procName := range procNames {
			procedure := dbSchema.Procedures[procName]
			// procName already contains signature as name(arguments) from inspector
			key := procedure.Schema + "." + procName
			newProcedures[key] = procedure
		}
	}

	// Find added procedures in deterministic order
	procedureKeys := sortedKeys(newProcedures)
	for _, key := range procedureKeys {
		procedure := newProcedures[key]
		if _, exists := oldProcedures[key]; !exists {
			diff.addedProcedures = append(diff.addedProcedures, procedure)
		}
	}

	// Find dropped procedures in deterministic order
	oldProcedureKeys := sortedKeys(oldProcedures)
	for _, key := range oldProcedureKeys {
		procedure := oldProcedures[key]
		if _, exists := newProcedures[key]; !exists {
			diff.droppedProcedures = append(diff.droppedProcedures, procedure)
		}
	}

	// Find modified procedures in deterministic order
	for _, key := range procedureKeys {
		newProcedure := newProcedures[key]
		if oldProcedure, exists := oldProcedures[key]; exists {
			if !proceduresEqual(oldProcedure, newProcedure) {
				diff.modifiedProcedures = append(diff.modifiedProcedures, &procedureDiff{
					Old: oldProcedure,
					New: newProcedure,
				})
			}
		}
	}

	// Compare types across all schemas
	oldTypes := make(map[string]*ir.Type)
	newTypes := make(map[string]*ir.Type)

	// Extract types from all schemas in oldIR in deterministic order
	for _, dbSchema := range oldIR.Schemas {
		typeNames := sortedKeys(dbSchema.Types)
		for _, typeName := range typeNames {
			typeObj := dbSchema.Types[typeName]
			key := typeObj.Schema + "." + typeName
			oldTypes[key] = typeObj
		}
	}

	// Extract types from all schemas in newIR in deterministic order
	for _, dbSchema := range newIR.Schemas {
		typeNames := sortedKeys(dbSchema.Types)
		for _, typeName := range typeNames {
			typeObj := dbSchema.Types[typeName]
			key := typeObj.Schema + "." + typeName
			newTypes[key] = typeObj
		}
	}

	// Find added types in deterministic order
	typeKeys := sortedKeys(newTypes)
	for _, key := range typeKeys {
		typeObj := newTypes[key]
		if _, exists := oldTypes[key]; !exists {
			diff.addedTypes = append(diff.addedTypes, typeObj)
		}
	}

	// Find dropped types in deterministic order
	oldTypeKeys := sortedKeys(oldTypes)
	for _, key := range oldTypeKeys {
		typeObj := oldTypes[key]
		if _, exists := newTypes[key]; !exists {
			diff.droppedTypes = append(diff.droppedTypes, typeObj)
		}
	}

	// Find modified types in deterministic order
	for _, key := range typeKeys {
		newType := newTypes[key]
		if oldType, exists := oldTypes[key]; exists {
			if !typesEqual(oldType, newType) {
				diff.modifiedTypes = append(diff.modifiedTypes, &typeDiff{
					Old: oldType,
					New: newType,
				})
			}
		}
	}

	// Compare views across all schemas
	oldViews := make(map[string]*ir.View)
	newViews := make(map[string]*ir.View)

	// Extract views from all schemas in oldIR in deterministic order
	for _, dbSchema := range oldIR.Schemas {
		viewNames := sortedKeys(dbSchema.Views)
		for _, viewName := range viewNames {
			view := dbSchema.Views[viewName]
			key := view.Schema + "." + viewName
			oldViews[key] = view
		}
	}

	// Extract views from all schemas in newIR in deterministic order
	for _, dbSchema := range newIR.Schemas {
		viewNames := sortedKeys(dbSchema.Views)
		for _, viewName := range viewNames {
			view := dbSchema.Views[viewName]
			key := view.Schema + "." + viewName
			newViews[key] = view
		}
	}

	// Find added views in deterministic order
	viewKeys := sortedKeys(newViews)
	for _, key := range viewKeys {
		view := newViews[key]
		if _, exists := oldViews[key]; !exists {
			diff.addedViews = append(diff.addedViews, view)
		}
	}

	// Find dropped views in deterministic order
	oldViewKeys := sortedKeys(oldViews)
	for _, key := range oldViewKeys {
		view := oldViews[key]
		if _, exists := newViews[key]; !exists {
			diff.droppedViews = append(diff.droppedViews, view)
		}
	}

	// Find modified views in deterministic order
	for _, key := range viewKeys {
		newView := newViews[key]
		if oldView, exists := oldViews[key]; exists {
			structurallyDifferent := !viewsEqual(oldView, newView)
			commentChanged := oldView.Comment != newView.Comment

			// Check if indexes changed for materialized views
			indexesChanged := false
			if newView.Materialized {
				oldIndexCount := 0
				newIndexCount := 0
				if oldView.Indexes != nil {
					oldIndexCount = len(oldView.Indexes)
				}
				if newView.Indexes != nil {
					newIndexCount = len(newView.Indexes)
				}
				indexesChanged = oldIndexCount != newIndexCount

				// If counts are same, check if any indexes are different (added/removed/modified)
				if !indexesChanged && oldIndexCount > 0 {
					// Check for added or removed indexes
					for indexName := range newView.Indexes {
						if _, exists := oldView.Indexes[indexName]; !exists {
							indexesChanged = true
							break
						}
					}

					// Check for modified indexes (structure or comments)
					if !indexesChanged {
						for indexName, newIndex := range newView.Indexes {
							if oldIndex, exists := oldView.Indexes[indexName]; exists {
								structurallyEqual := indexesStructurallyEqual(oldIndex, newIndex)
								commentChanged := oldIndex.Comment != newIndex.Comment
								if !structurallyEqual || commentChanged {
									indexesChanged = true
									break
								}
							}
						}
					}
				}
			}

			if structurallyDifferent || commentChanged || indexesChanged {
				// For materialized views with structural changes, mark for recreation
				if newView.Materialized && structurallyDifferent {
					diff.modifiedViews = append(diff.modifiedViews, &viewDiff{
						Old:              oldView,
						New:              newView,
						RequiresRecreate: true,
					})
				} else {
					// For regular views or comment-only changes, use the modify approach
					viewDiff := &viewDiff{
						Old: oldView,
						New: newView,
					}

					// Check for comment changes
					if commentChanged {
						viewDiff.CommentChanged = true
						viewDiff.OldComment = oldView.Comment
						viewDiff.NewComment = newView.Comment
					}

					// For materialized views, also diff indexes
					if newView.Materialized {
						oldIndexes := oldView.Indexes
						newIndexes := newView.Indexes
						if oldIndexes == nil {
							oldIndexes = make(map[string]*ir.Index)
						}
						if newIndexes == nil {
							newIndexes = make(map[string]*ir.Index)
						}

						// Find added indexes
						for indexName, index := range newIndexes {
							if _, exists := oldIndexes[indexName]; !exists {
								viewDiff.AddedIndexes = append(viewDiff.AddedIndexes, index)
							}
						}

						// Find dropped indexes
						for indexName, index := range oldIndexes {
							if _, exists := newIndexes[indexName]; !exists {
								viewDiff.DroppedIndexes = append(viewDiff.DroppedIndexes, index)
							}
						}

						// Find modified indexes
						for indexName, newIndex := range newIndexes {
							if oldIndex, exists := oldIndexes[indexName]; exists {
								structurallyEqual := indexesStructurallyEqual(oldIndex, newIndex)
								commentChanged := oldIndex.Comment != newIndex.Comment

								// If either structure changed or comment changed, treat as modification
								if !structurallyEqual || commentChanged {
									viewDiff.ModifiedIndexes = append(viewDiff.ModifiedIndexes, &IndexDiff{
										Old: oldIndex,
										New: newIndex,
									})
								}
							}
						}
					}

					diff.modifiedViews = append(diff.modifiedViews, viewDiff)
				}
			}
		}
	}

	// Compare sequences across all schemas
	oldSequences := make(map[string]*ir.Sequence)
	newSequences := make(map[string]*ir.Sequence)

	// Extract sequences from all schemas in oldIR in deterministic order
	for _, dbSchema := range oldIR.Schemas {
		seqNames := sortedKeys(dbSchema.Sequences)
		for _, seqName := range seqNames {
			seq := dbSchema.Sequences[seqName]
			key := seq.Schema + "." + seqName
			oldSequences[key] = seq
		}
	}

	// Extract sequences from all schemas in newIR in deterministic order
	for _, dbSchema := range newIR.Schemas {
		seqNames := sortedKeys(dbSchema.Sequences)
		for _, seqName := range seqNames {
			seq := dbSchema.Sequences[seqName]
			key := seq.Schema + "." + seqName
			newSequences[key] = seq
		}
	}

	// Find added sequences in deterministic order
	seqKeys := sortedKeys(newSequences)
	for _, key := range seqKeys {
		seq := newSequences[key]
		if _, exists := oldSequences[key]; !exists {
			// Skip sequences owned by table columns (created by SERIAL)
			if seq.OwnedByTable != "" && seq.OwnedByColumn != "" {
				continue
			}
			diff.addedSequences = append(diff.addedSequences, seq)
		}
	}

	// Find dropped sequences in deterministic order
	oldSeqKeys := sortedKeys(oldSequences)
	for _, key := range oldSeqKeys {
		seq := oldSequences[key]
		if _, exists := newSequences[key]; !exists {
			// Skip sequences owned by table columns (created by SERIAL)
			if seq.OwnedByTable != "" && seq.OwnedByColumn != "" {
				continue
			}
			diff.droppedSequences = append(diff.droppedSequences, seq)
		}
	}

	// Find modified sequences in deterministic order
	for _, key := range seqKeys {
		newSeq := newSequences[key]
		if oldSeq, exists := oldSequences[key]; exists {
			// Skip sequences owned by table columns (created by SERIAL)
			if (oldSeq.OwnedByTable != "" && oldSeq.OwnedByColumn != "") ||
				(newSeq.OwnedByTable != "" && newSeq.OwnedByColumn != "") {
				continue
			}
			if !sequencesEqual(oldSeq, newSeq) {
				diff.modifiedSequences = append(diff.modifiedSequences, &sequenceDiff{
					Old: oldSeq,
					New: newSeq,
				})
			}
		}
	}

	// Compare default privileges across all schemas
	oldDefaultPrivs := make(map[string]*ir.DefaultPrivilege)
	newDefaultPrivs := make(map[string]*ir.DefaultPrivilege)

	// Extract default privileges from all schemas in oldIR
	for _, dbSchema := range oldIR.Schemas {
		for _, dp := range dbSchema.DefaultPrivileges {
			key := dp.OwnerRole + ":" + string(dp.ObjectType) + ":" + dp.Grantee
			oldDefaultPrivs[key] = dp
		}
	}

	// Extract default privileges from all schemas in newIR
	for _, dbSchema := range newIR.Schemas {
		for _, dp := range dbSchema.DefaultPrivileges {
			key := dp.OwnerRole + ":" + string(dp.ObjectType) + ":" + dp.Grantee
			newDefaultPrivs[key] = dp
		}
	}

	// Find added default privileges
	for key, dp := range newDefaultPrivs {
		if _, exists := oldDefaultPrivs[key]; !exists {
			diff.addedDefaultPrivileges = append(diff.addedDefaultPrivileges, dp)
		}
	}

	// Find dropped default privileges
	for key, dp := range oldDefaultPrivs {
		if _, exists := newDefaultPrivs[key]; !exists {
			diff.droppedDefaultPrivileges = append(diff.droppedDefaultPrivileges, dp)
		}
	}

	// Find modified default privileges
	for key, newDP := range newDefaultPrivs {
		if oldDP, exists := oldDefaultPrivs[key]; exists {
			if !defaultPrivilegesEqual(oldDP, newDP) {
				diff.modifiedDefaultPrivileges = append(diff.modifiedDefaultPrivileges, &defaultPrivilegeDiff{
					Old: oldDP,
					New: newDP,
				})
			}
		}
	}

	// Sort default privileges for deterministic output (by owner_role, then object_type, then grantee)
	sort.Slice(diff.addedDefaultPrivileges, func(i, j int) bool {
		if diff.addedDefaultPrivileges[i].OwnerRole != diff.addedDefaultPrivileges[j].OwnerRole {
			return diff.addedDefaultPrivileges[i].OwnerRole < diff.addedDefaultPrivileges[j].OwnerRole
		}
		if diff.addedDefaultPrivileges[i].ObjectType != diff.addedDefaultPrivileges[j].ObjectType {
			return diff.addedDefaultPrivileges[i].ObjectType < diff.addedDefaultPrivileges[j].ObjectType
		}
		return diff.addedDefaultPrivileges[i].Grantee < diff.addedDefaultPrivileges[j].Grantee
	})
	sort.Slice(diff.droppedDefaultPrivileges, func(i, j int) bool {
		if diff.droppedDefaultPrivileges[i].OwnerRole != diff.droppedDefaultPrivileges[j].OwnerRole {
			return diff.droppedDefaultPrivileges[i].OwnerRole < diff.droppedDefaultPrivileges[j].OwnerRole
		}
		if diff.droppedDefaultPrivileges[i].ObjectType != diff.droppedDefaultPrivileges[j].ObjectType {
			return diff.droppedDefaultPrivileges[i].ObjectType < diff.droppedDefaultPrivileges[j].ObjectType
		}
		return diff.droppedDefaultPrivileges[i].Grantee < diff.droppedDefaultPrivileges[j].Grantee
	})
	sort.Slice(diff.modifiedDefaultPrivileges, func(i, j int) bool {
		if diff.modifiedDefaultPrivileges[i].New.OwnerRole != diff.modifiedDefaultPrivileges[j].New.OwnerRole {
			return diff.modifiedDefaultPrivileges[i].New.OwnerRole < diff.modifiedDefaultPrivileges[j].New.OwnerRole
		}
		if diff.modifiedDefaultPrivileges[i].New.ObjectType != diff.modifiedDefaultPrivileges[j].New.ObjectType {
			return diff.modifiedDefaultPrivileges[i].New.ObjectType < diff.modifiedDefaultPrivileges[j].New.ObjectType
		}
		return diff.modifiedDefaultPrivileges[i].New.Grantee < diff.modifiedDefaultPrivileges[j].New.Grantee
	})

	// Compare explicit object privileges across all schemas
	// Use GetFullKey() to avoid overwrites when same (object, grantee) has different grant options
	oldPrivs := make(map[string]*ir.Privilege)
	newPrivs := make(map[string]*ir.Privilege)

	for _, dbSchema := range oldIR.Schemas {
		for _, p := range dbSchema.Privileges {
			key := p.GetFullKey()
			oldPrivs[key] = p
		}
	}

	for _, dbSchema := range newIR.Schemas {
		for _, p := range dbSchema.Privileges {
			key := p.GetFullKey()
			newPrivs[key] = p
		}
	}

	// Build index by GetObjectKey() to find matching privileges for modification detection
	oldPrivsByObjectKey := make(map[string][]*ir.Privilege)
	newPrivsByObjectKey := make(map[string][]*ir.Privilege)
	for _, p := range oldPrivs {
		key := p.GetObjectKey()
		oldPrivsByObjectKey[key] = append(oldPrivsByObjectKey[key], p)
	}
	for _, p := range newPrivs {
		key := p.GetObjectKey()
		newPrivsByObjectKey[key] = append(newPrivsByObjectKey[key], p)
	}

	// Track which privileges have been matched for modification
	matchedOld := make(map[string]bool)
	matchedNew := make(map[string]bool)

	// Find modified privileges - match by GetObjectKey() to detect grant option changes
	for objectKey, newList := range newPrivsByObjectKey {
		oldList := oldPrivsByObjectKey[objectKey]
		if len(oldList) == 0 {
			continue
		}

		// Simple case: one privilege each, check for modification
		if len(oldList) == 1 && len(newList) == 1 {
			oldP, newP := oldList[0], newList[0]
			if !privilegesEqual(oldP, newP) {
				diff.modifiedPrivileges = append(diff.modifiedPrivileges, &privilegeDiff{
					Old: oldP,
					New: newP,
				})
			}
			matchedOld[oldP.GetFullKey()] = true
			matchedNew[newP.GetFullKey()] = true
			continue
		}

		// Complex case: multiple privileges with same object key but different grant options
		// Match by full key first, then handle remaining as add/drop
		for _, newP := range newList {
			fullKey := newP.GetFullKey()
			if oldP, exists := oldPrivs[fullKey]; exists {
				if !privilegesEqual(oldP, newP) {
					diff.modifiedPrivileges = append(diff.modifiedPrivileges, &privilegeDiff{
						Old: oldP,
						New: newP,
					})
				}
				matchedOld[fullKey] = true
				matchedNew[fullKey] = true
			}
		}
	}

	// Find added privileges (in new but not matched)
	for fullKey, p := range newPrivs {
		if !matchedNew[fullKey] {
			diff.addedPrivileges = append(diff.addedPrivileges, p)
		}
	}

	// Find dropped privileges (in old but not matched)
	for fullKey, p := range oldPrivs {
		if !matchedOld[fullKey] {
			diff.droppedPrivileges = append(diff.droppedPrivileges, p)
		}
	}

	// Sort privileges for deterministic output
	sort.Slice(diff.addedPrivileges, func(i, j int) bool {
		return diff.addedPrivileges[i].GetObjectKey() < diff.addedPrivileges[j].GetObjectKey()
	})
	sort.Slice(diff.droppedPrivileges, func(i, j int) bool {
		return diff.droppedPrivileges[i].GetObjectKey() < diff.droppedPrivileges[j].GetObjectKey()
	})
	sort.Slice(diff.modifiedPrivileges, func(i, j int) bool {
		return diff.modifiedPrivileges[i].New.GetObjectKey() < diff.modifiedPrivileges[j].New.GetObjectKey()
	})

	// Compare revoked default privileges across all schemas
	oldRevokedPrivs := make(map[string]*ir.RevokedDefaultPrivilege)
	newRevokedPrivs := make(map[string]*ir.RevokedDefaultPrivilege)

	for _, dbSchema := range oldIR.Schemas {
		for _, r := range dbSchema.RevokedDefaultPrivileges {
			key := r.GetObjectKey()
			oldRevokedPrivs[key] = r
		}
	}

	for _, dbSchema := range newIR.Schemas {
		for _, r := range dbSchema.RevokedDefaultPrivileges {
			key := r.GetObjectKey()
			newRevokedPrivs[key] = r
		}
	}

	// Find added revoked default privileges (new revokes)
	for key, r := range newRevokedPrivs {
		if _, exists := oldRevokedPrivs[key]; !exists {
			diff.addedRevokedDefaultPrivs = append(diff.addedRevokedDefaultPrivs, r)
		}
	}

	// Find dropped revoked default privileges (restored defaults)
	for key, r := range oldRevokedPrivs {
		if _, exists := newRevokedPrivs[key]; !exists {
			diff.droppedRevokedDefaultPrivs = append(diff.droppedRevokedDefaultPrivs, r)
		}
	}

	// Sort revoked default privileges for deterministic output
	sort.Slice(diff.addedRevokedDefaultPrivs, func(i, j int) bool {
		return diff.addedRevokedDefaultPrivs[i].GetObjectKey() < diff.addedRevokedDefaultPrivs[j].GetObjectKey()
	})
	sort.Slice(diff.droppedRevokedDefaultPrivs, func(i, j int) bool {
		return diff.droppedRevokedDefaultPrivs[i].GetObjectKey() < diff.droppedRevokedDefaultPrivs[j].GetObjectKey()
	})

	// Sort tables and views topologically for consistent ordering
	// Pre-sort by name to ensure deterministic insertion order for cycle breaking
	sort.Slice(diff.addedTables, func(i, j int) bool {
		return diff.addedTables[i].Schema+"."+diff.addedTables[i].Name < diff.addedTables[j].Schema+"."+diff.addedTables[j].Name
	})
	diff.addedTables = topologicallySortTables(diff.addedTables)

	sort.Slice(diff.droppedTables, func(i, j int) bool {
		return diff.droppedTables[i].Schema+"."+diff.droppedTables[i].Name < diff.droppedTables[j].Schema+"."+diff.droppedTables[j].Name
	})
	diff.droppedTables = reverseSlice(topologicallySortTables(diff.droppedTables))
	diff.addedViews = topologicallySortViews(diff.addedViews)
	diff.droppedViews = reverseSlice(topologicallySortViews(diff.droppedViews))

	// Sort ModifiedTables alphabetically for consistent ordering
	// (topological sorting isn't needed for modified tables since they already exist)
	sortModifiedTables(diff.modifiedTables)

	// Sort individual table objects (indexes, triggers, policies, constraints) within each table
	sortTableObjects(diff.modifiedTables)

	// Create a diffCollector and generate SQL
	collector := newDiffCollector()
	diff.collectMigrationSQL(targetSchema, collector)
	return collector.diffs
}

// collectMigrationSQL populates the collector with SQL statements for the diff
// The collector must not be nil
func (d *ddlDiff) collectMigrationSQL(targetSchema string, collector *diffCollector) {
	// Pre-drop materialized views that depend on tables being modified/dropped
	// This must happen BEFORE table operations to avoid dependency errors
	preDroppedViews := d.generatePreDropMaterializedViewsSQL(targetSchema, collector)

	// First: Drop operations (in reverse dependency order)
	d.generateDropSQL(targetSchema, collector, preDroppedViews)

	// Then: Create operations (in dependency order)
	d.generateCreateSQL(targetSchema, collector)

	// Finally: Modify operations
	d.generateModifySQL(targetSchema, collector, preDroppedViews)
}

// generatePreDropMaterializedViewsSQL drops materialized views that depend on
// tables being modified or dropped. This must happen before table operations.
// Returns a set of pre-dropped view keys (schema.name) to avoid duplicate drops.
func (d *ddlDiff) generatePreDropMaterializedViewsSQL(targetSchema string, collector *diffCollector) map[string]bool {
	preDropped := make(map[string]bool)

	// Build set of tables being modified or dropped
	affectedTables := make(map[string]*ir.Table)
	for _, table := range d.droppedTables {
		key := table.Schema + "." + table.Name
		affectedTables[key] = table
	}
	for _, tableDiff := range d.modifiedTables {
		key := tableDiff.Table.Schema + "." + tableDiff.Table.Name
		affectedTables[key] = tableDiff.Table
	}

	if len(affectedTables) == 0 {
		return preDropped
	}

	// Check modifiedViews with RequiresRecreate for dependencies on affected tables
	for _, viewDiff := range d.modifiedViews {
		if !viewDiff.RequiresRecreate || !viewDiff.New.Materialized {
			continue
		}

		viewKey := viewDiff.Old.Schema + "." + viewDiff.Old.Name
		if preDropped[viewKey] {
			continue
		}

		// Check if this view depends on any affected table
		for _, table := range affectedTables {
			if viewDependsOnTable(viewDiff.Old, table.Schema, table.Name) {
				// Pre-drop this materialized view (it will be recreated later)
				viewName := qualifyEntityName(viewDiff.Old.Schema, viewDiff.Old.Name, targetSchema)
				sql := fmt.Sprintf("DROP MATERIALIZED VIEW %s RESTRICT;", viewName)

				// Use DiffOperationRecreate to indicate this DROP is part of a
				// modification cycle - the view will be recreated after table changes
				context := &diffContext{
					Type:                DiffTypeMaterializedView,
					Operation:           DiffOperationRecreate,
					Path:                fmt.Sprintf("%s.%s", viewDiff.Old.Schema, viewDiff.Old.Name),
					Source:              viewDiff.Old,
					CanRunInTransaction: true,
				}
				collector.collect(context, sql)

				preDropped[viewKey] = true
				break
			}
		}
	}

	// Check droppedViews for dependencies on dropped tables (for CASCADE scenario)
	for _, view := range d.droppedViews {
		if !view.Materialized {
			continue
		}

		viewKey := view.Schema + "." + view.Name
		if preDropped[viewKey] {
			continue
		}

		// Check if this view depends on any dropped table
		for _, table := range d.droppedTables {
			if viewDependsOnTable(view, table.Schema, table.Name) {
				// Pre-drop this materialized view before the table CASCADE
				viewName := qualifyEntityName(view.Schema, view.Name, targetSchema)
				sql := fmt.Sprintf("DROP MATERIALIZED VIEW %s RESTRICT;", viewName)

				context := &diffContext{
					Type:                DiffTypeMaterializedView,
					Operation:           DiffOperationDrop,
					Path:                fmt.Sprintf("%s.%s", view.Schema, view.Name),
					Source:              view,
					CanRunInTransaction: true,
				}
				collector.collect(context, sql)

				preDropped[viewKey] = true
				break
			}
		}
	}

	return preDropped
}

// generateCreateSQL generates CREATE statements in dependency order
func (d *ddlDiff) generateCreateSQL(targetSchema string, collector *diffCollector) {
	// Note: Schema creation is out of scope for schema-level comparisons

	// Create types
	generateCreateTypesSQL(d.addedTypes, targetSchema, collector)

	// Create sequences
	generateCreateSequencesSQL(d.addedSequences, targetSchema, collector)

	// Build map of existing tables (tables being modified, so they already exist)
	existingTables := make(map[string]bool, len(d.modifiedTables))
	for _, tableDiff := range d.modifiedTables {
		key := fmt.Sprintf("%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name)
		existingTables[key] = true
	}

	newFunctionLookup := buildFunctionLookup(d.addedFunctions)
	var shouldDeferPolicy func(*ir.RLSPolicy) bool
	if len(newFunctionLookup) > 0 {
		shouldDeferPolicy = func(policy *ir.RLSPolicy) bool {
			return policyReferencesNewFunction(policy, newFunctionLookup)
		}
	}

	// Separate tables into those that depend on new functions and those that don't
	// This ensures we create functions before tables that use them in defaults/checks
	tablesWithoutFunctionDeps := []*ir.Table{}
	tablesWithFunctionDeps := []*ir.Table{}

	for _, table := range d.addedTables {
		if tableReferencesNewFunction(table, newFunctionLookup) {
			tablesWithFunctionDeps = append(tablesWithFunctionDeps, table)
		} else {
			tablesWithoutFunctionDeps = append(tablesWithoutFunctionDeps, table)
		}
	}

	// Create tables WITHOUT function dependencies first (functions may reference these)
	deferredPolicies1, deferredConstraints1 := generateCreateTablesSQL(tablesWithoutFunctionDeps, targetSchema, collector, existingTables, shouldDeferPolicy)

	// Create functions (functions may depend on tables created above)
	generateCreateFunctionsSQL(d.addedFunctions, targetSchema, collector)

	// Create procedures (procedures may depend on tables)
	generateCreateProceduresSQL(d.addedProcedures, targetSchema, collector)

	// Create tables WITH function dependencies (now that functions exist)
	deferredPolicies2, deferredConstraints2 := generateCreateTablesSQL(tablesWithFunctionDeps, targetSchema, collector, existingTables, shouldDeferPolicy)

	// Add deferred foreign key constraints from BOTH batches AFTER all tables are created
	// This ensures FK references to tables in the second batch (function-dependent tables) work correctly
	allDeferredConstraints := append(deferredConstraints1, deferredConstraints2...)
	generateDeferredConstraintsSQL(allDeferredConstraints, targetSchema, collector)

	// Merge deferred policies from both batches
	allDeferredPolicies := append(deferredPolicies1, deferredPolicies2...)

	// Create policies after functions/procedures to satisfy dependencies
	generateCreatePoliciesSQL(allDeferredPolicies, targetSchema, collector)

	// Create triggers (triggers may depend on functions/procedures)
	// Note: We need to create triggers for ALL tables, not just the original d.addedTables
	generateCreateTriggersFromTables(d.addedTables, targetSchema, collector)

	// Create views
	generateCreateViewsSQL(d.addedViews, targetSchema, collector)

	// Create default privileges
	generateCreateDefaultPrivilegesSQL(d.addedDefaultPrivileges, targetSchema, collector)

	// Create explicit object privileges
	generateCreatePrivilegesSQL(d.addedPrivileges, targetSchema, collector)

	// Revoke default PUBLIC privileges (new revokes)
	generateRevokeDefaultPrivilegesSQL(d.addedRevokedDefaultPrivs, targetSchema, collector)
}

// generateModifySQL generates ALTER statements
// preDroppedViews contains views that were already dropped in the pre-drop phase
func (d *ddlDiff) generateModifySQL(targetSchema string, collector *diffCollector, preDroppedViews map[string]bool) {
	// Modify schemas
	// Note: Schema modification is out of scope for schema-level comparisons

	// Modify types
	generateModifyTypesSQL(d.modifiedTypes, targetSchema, collector)

	// Modify sequences
	generateModifySequencesSQL(d.modifiedSequences, targetSchema, collector)

	// Modify tables
	generateModifyTablesSQL(d.modifiedTables, targetSchema, collector)

	// Modify views - pass preDroppedViews to skip DROP for already-dropped views
	generateModifyViewsSQL(d.modifiedViews, targetSchema, collector, preDroppedViews)

	// Modify functions
	generateModifyFunctionsSQL(d.modifiedFunctions, targetSchema, collector)

	// Modify procedures
	generateModifyProceduresSQL(d.modifiedProcedures, targetSchema, collector)

	// Modify default privileges
	generateModifyDefaultPrivilegesSQL(d.modifiedDefaultPrivileges, targetSchema, collector)

	// Modify explicit object privileges
	generateModifyPrivilegesSQL(d.modifiedPrivileges, targetSchema, collector)
}

// generateDropSQL generates DROP statements in reverse dependency order
// preDroppedViews contains views that were already dropped in the pre-drop phase
func (d *ddlDiff) generateDropSQL(targetSchema string, collector *diffCollector, preDroppedViews map[string]bool) {

	// Drop triggers from modified tables first (triggers depend on functions)
	generateDropTriggersFromModifiedTables(d.modifiedTables, targetSchema, collector)

	// Drop functions
	generateDropFunctionsSQL(d.droppedFunctions, targetSchema, collector)

	// Drop procedures
	generateDropProceduresSQL(d.droppedProcedures, targetSchema, collector)

	// Drop views - filter out pre-dropped ones to avoid duplicate drops
	viewsToDrop := filterPreDroppedViews(d.droppedViews, preDroppedViews)
	generateDropViewsSQL(viewsToDrop, targetSchema, collector)

	// Drop tables
	generateDropTablesSQL(d.droppedTables, targetSchema, collector)

	// Drop sequences
	generateDropSequencesSQL(d.droppedSequences, targetSchema, collector)

	// Drop types
	generateDropTypesSQL(d.droppedTypes, targetSchema, collector)

	// Restore default PUBLIC privileges (dropped revokes = restore defaults)
	generateRestoreDefaultPrivilegesSQL(d.droppedRevokedDefaultPrivs, targetSchema, collector)

	// Drop explicit object privileges
	generateDropPrivilegesSQL(d.droppedPrivileges, targetSchema, collector)

	// Drop default privileges
	generateDropDefaultPrivilegesSQL(d.droppedDefaultPrivileges, targetSchema, collector)

	// Drop schemas
	// Note: Schema deletion is out of scope for schema-level comparisons
}

// filterPreDroppedViews returns views that haven't been pre-dropped
func filterPreDroppedViews(views []*ir.View, preDropped map[string]bool) []*ir.View {
	if len(preDropped) == 0 {
		return views
	}

	filtered := make([]*ir.View, 0, len(views))
	for _, view := range views {
		key := view.Schema + "." + view.Name
		if !preDropped[key] {
			filtered = append(filtered, view)
		}
	}
	return filtered
}

// getTableNameWithSchema returns the table name with schema qualification only when necessary
// If the table schema is different from the target schema, it returns "schema.table"
// If they are the same, it returns just "table"
func getTableNameWithSchema(tableSchema, tableName, targetSchema string) string {
	quotedTable := ir.QuoteIdentifier(tableName)
	if tableSchema != targetSchema {
		quotedSchema := ir.QuoteIdentifier(tableSchema)
		return fmt.Sprintf("%s.%s", quotedSchema, quotedTable)
	}
	return quotedTable
}

// qualifyEntityName returns the properly qualified entity name based on target schema
// If entity is in target schema, returns just the name, otherwise returns schema.name
func qualifyEntityName(entitySchema, entityName, targetSchema string) string {
	quotedName := ir.QuoteIdentifier(entityName)
	if entitySchema == targetSchema {
		return quotedName
	}
	quotedSchema := ir.QuoteIdentifier(entitySchema)
	return fmt.Sprintf("%s.%s", quotedSchema, quotedName)
}

// quoteString properly quotes a string for SQL, handling single quotes
func quoteString(s string) string {
	// Escape single quotes by doubling them
	escaped := strings.ReplaceAll(s, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}

// sortModifiedTables sorts modified tables alphabetically by schema then name
func sortModifiedTables(tables []*tableDiff) {
	sort.Slice(tables, func(i, j int) bool {
		// First sort by schema, then by table name
		if tables[i].Table.Schema != tables[j].Table.Schema {
			return tables[i].Table.Schema < tables[j].Table.Schema
		}
		return tables[i].Table.Name < tables[j].Table.Name
	})
}

// sortTableObjects sorts the objects within each table diff for consistent ordering
func sortTableObjects(tables []*tableDiff) {
	for _, tableDiff := range tables {
		// Sort dropped constraints
		sort.Slice(tableDiff.DroppedConstraints, func(i, j int) bool {
			return tableDiff.DroppedConstraints[i].Name < tableDiff.DroppedConstraints[j].Name
		})

		// Sort added constraints
		sort.Slice(tableDiff.AddedConstraints, func(i, j int) bool {
			return tableDiff.AddedConstraints[i].Name < tableDiff.AddedConstraints[j].Name
		})

		// Sort modified constraints
		sort.Slice(tableDiff.ModifiedConstraints, func(i, j int) bool {
			return tableDiff.ModifiedConstraints[i].New.Name < tableDiff.ModifiedConstraints[j].New.Name
		})

		// Sort dropped policies
		sort.Slice(tableDiff.DroppedPolicies, func(i, j int) bool {
			return tableDiff.DroppedPolicies[i].Name < tableDiff.DroppedPolicies[j].Name
		})

		// Sort added policies
		sort.Slice(tableDiff.AddedPolicies, func(i, j int) bool {
			return tableDiff.AddedPolicies[i].Name < tableDiff.AddedPolicies[j].Name
		})

		// Sort modified policies
		sort.Slice(tableDiff.ModifiedPolicies, func(i, j int) bool {
			return tableDiff.ModifiedPolicies[i].New.Name < tableDiff.ModifiedPolicies[j].New.Name
		})

		// Sort dropped triggers
		sort.Slice(tableDiff.DroppedTriggers, func(i, j int) bool {
			return tableDiff.DroppedTriggers[i].Name < tableDiff.DroppedTriggers[j].Name
		})

		// Sort added triggers
		sort.Slice(tableDiff.AddedTriggers, func(i, j int) bool {
			return tableDiff.AddedTriggers[i].Name < tableDiff.AddedTriggers[j].Name
		})

		// Sort modified triggers
		sort.Slice(tableDiff.ModifiedTriggers, func(i, j int) bool {
			return tableDiff.ModifiedTriggers[i].New.Name < tableDiff.ModifiedTriggers[j].New.Name
		})

		// Sort dropped indexes
		sort.Slice(tableDiff.DroppedIndexes, func(i, j int) bool {
			return tableDiff.DroppedIndexes[i].Name < tableDiff.DroppedIndexes[j].Name
		})

		// Sort added indexes
		sort.Slice(tableDiff.AddedIndexes, func(i, j int) bool {
			return tableDiff.AddedIndexes[i].Name < tableDiff.AddedIndexes[j].Name
		})

		// Sort modified indexes
		sort.Slice(tableDiff.ModifiedIndexes, func(i, j int) bool {
			return tableDiff.ModifiedIndexes[i].New.Name < tableDiff.ModifiedIndexes[j].New.Name
		})

		// Sort columns by position for consistent ordering
		sort.Slice(tableDiff.DroppedColumns, func(i, j int) bool {
			return tableDiff.DroppedColumns[i].Position < tableDiff.DroppedColumns[j].Position
		})

		sort.Slice(tableDiff.AddedColumns, func(i, j int) bool {
			return tableDiff.AddedColumns[i].Position < tableDiff.AddedColumns[j].Position
		})

		sort.Slice(tableDiff.ModifiedColumns, func(i, j int) bool {
			return tableDiff.ModifiedColumns[i].New.Position < tableDiff.ModifiedColumns[j].New.Position
		})
	}
}

// sortedKeys returns sorted keys from a map[string]T
func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// buildFunctionLookup returns case-insensitive lookup keys for newly added functions.
// Keys include both unqualified (function name only) and schema-qualified identifiers.
func buildFunctionLookup(functions []*ir.Function) map[string]struct{} {
	if len(functions) == 0 {
		return nil
	}

	lookup := make(map[string]struct{}, len(functions)*2)
	for _, fn := range functions {
		if fn == nil || fn.Name == "" {
			continue
		}

		name := strings.ToLower(fn.Name)
		lookup[name] = struct{}{}

		if fn.Schema != "" {
			qualified := fmt.Sprintf("%s.%s", strings.ToLower(fn.Schema), name)
			lookup[qualified] = struct{}{}
		}
	}
	return lookup
}

var functionCallRegex = regexp.MustCompile(`(?i)([a-z_][a-z0-9_$]*(?:\.[a-z_][a-z0-9_$]*)*)\s*\(`)

// tableReferencesNewFunction determines if a table references any newly added functions
// in column defaults, generated columns, or CHECK constraints.
func tableReferencesNewFunction(table *ir.Table, newFunctions map[string]struct{}) bool {
	if len(newFunctions) == 0 || table == nil {
		return false
	}

	// Check column defaults and generated expressions
	for _, col := range table.Columns {
		// Check default value
		if col.DefaultValue != nil && *col.DefaultValue != "" {
			if referencesNewFunction(*col.DefaultValue, table.Schema, newFunctions) {
				return true
			}
		}
		// Check generated column expression
		if col.GeneratedExpr != nil && *col.GeneratedExpr != "" {
			if referencesNewFunction(*col.GeneratedExpr, table.Schema, newFunctions) {
				return true
			}
		}
	}

	// Check CHECK constraints
	for _, constraint := range table.Constraints {
		if constraint.Type == ir.ConstraintTypeCheck && constraint.CheckClause != "" {
			if referencesNewFunction(constraint.CheckClause, table.Schema, newFunctions) {
				return true
			}
		}
	}

	return false
}

// policyReferencesNewFunction determines if a policy references any newly added functions.
func policyReferencesNewFunction(policy *ir.RLSPolicy, newFunctions map[string]struct{}) bool {
	if len(newFunctions) == 0 || policy == nil {
		return false
	}

	for _, expr := range []string{policy.Using, policy.WithCheck} {
		if referencesNewFunction(expr, policy.Schema, newFunctions) {
			return true
		}
	}
	return false
}

func referencesNewFunction(expr, defaultSchema string, newFunctions map[string]struct{}) bool {
	if expr == "" || len(newFunctions) == 0 {
		return false
	}

	matches := functionCallRegex.FindAllStringSubmatch(expr, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		identifier := strings.ToLower(match[1])
		if identifier == "" {
			continue
		}

		if _, ok := newFunctions[identifier]; ok {
			return true
		}

		if !strings.Contains(identifier, ".") && defaultSchema != "" {
			qualified := fmt.Sprintf("%s.%s", strings.ToLower(defaultSchema), identifier)
			if _, ok := newFunctions[qualified]; ok {
				return true
			}
		}
	}
	return false
}

// GetObjectName implementations for DiffSource interface
func (d *schemaDiff) GetObjectName() string     { return d.New.Name }
func (d *functionDiff) GetObjectName() string   { return d.New.Name }
func (d *procedureDiff) GetObjectName() string  { return d.New.Name }
func (d *typeDiff) GetObjectName() string       { return d.New.Name }
func (d *sequenceDiff) GetObjectName() string   { return d.New.Name }
func (d *triggerDiff) GetObjectName() string    { return d.New.Name }
func (d *viewDiff) GetObjectName() string       { return d.New.Name }
func (d *tableDiff) GetObjectName() string      { return d.Table.Name }
func (d *ColumnDiff) GetObjectName() string     { return d.New.Name }
func (d *ConstraintDiff) GetObjectName() string { return d.New.Name }
func (d *IndexDiff) GetObjectName() string      { return d.New.Name }
func (d *policyDiff) GetObjectName() string     { return d.New.Name }
func (d *rlsChange) GetObjectName() string      { return d.Table.Name }
