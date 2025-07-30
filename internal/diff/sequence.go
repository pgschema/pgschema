package diff

import (
	"fmt"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// generateCreateSequencesSQL generates CREATE SEQUENCE statements
func generateCreateSequencesSQL(w Writer, sequences []*ir.Sequence, targetSchema string) {
	for _, seq := range sequences {
		w.WriteDDLSeparator()
		sql := generateSequenceSQL(seq, targetSchema)
		w.WriteStatementWithComment("SEQUENCE", seq.Name, seq.Schema, "", sql, targetSchema)
	}
}

// generateDropSequencesSQL generates DROP SEQUENCE statements
func generateDropSequencesSQL(w Writer, sequences []*ir.Sequence, targetSchema string) {
	// Process sequences in reverse order (already sorted)
	for _, seq := range sequences {
		w.WriteDDLSeparator()
		seqName := qualifyEntityName(seq.Schema, seq.Name, targetSchema)
		sql := fmt.Sprintf("DROP SEQUENCE IF EXISTS %s CASCADE;", seqName)
		w.WriteStatementWithComment("SEQUENCE", seq.Name, seq.Schema, "", sql, targetSchema)
	}
}

// generateModifySequencesSQL generates ALTER SEQUENCE statements
func generateModifySequencesSQL(w Writer, diffs []*SequenceDiff, targetSchema string) {
	for _, diff := range diffs {
		statements := diff.generateAlterSequenceStatements(targetSchema)
		for _, stmt := range statements {
			w.WriteDDLSeparator()
			w.WriteStatementWithComment("SEQUENCE", diff.New.Name, diff.New.Schema, "", stmt, targetSchema)
		}
	}
}

// generateSequenceSQL generates CREATE SEQUENCE statement
func generateSequenceSQL(seq *ir.Sequence, targetSchema string) string {
	var parts []string
	
	seqName := qualifyEntityName(seq.Schema, seq.Name, targetSchema)
	parts = append(parts, fmt.Sprintf("CREATE SEQUENCE %s", seqName))
	
	// Add sequence parameters if they differ from defaults
	// Always include START WITH if it's not 1, but also include it if we have an explicit start value
	if seq.StartValue != 1 {
		parts = append(parts, fmt.Sprintf("START WITH %d", seq.StartValue))
	}
	
	if seq.Increment != 1 {
		parts = append(parts, fmt.Sprintf("INCREMENT BY %d", seq.Increment))
	}
	
	if seq.MinValue != nil && *seq.MinValue != 1 {
		parts = append(parts, fmt.Sprintf("MINVALUE %d", *seq.MinValue))
	}
	
	if seq.MaxValue != nil && *seq.MaxValue != 9223372036854775807 { // Default BIGINT max
		parts = append(parts, fmt.Sprintf("MAXVALUE %d", *seq.MaxValue))
	}
	
	// Cache is not part of ir.Sequence, skip it
	
	if seq.CycleOption {
		parts = append(parts, "CYCLE")
	}
	
	// Join with proper formatting
	if len(parts) > 1 {
		return parts[0] + " " + strings.Join(parts[1:], " ") + ";"
	}
	return parts[0] + ";"
}

// generateAlterSequenceStatements generates ALTER SEQUENCE statements for modifications
func (d *SequenceDiff) generateAlterSequenceStatements(targetSchema string) []string {
	var statements []string
	
	seqName := qualifyEntityName(d.New.Schema, d.New.Name, targetSchema)
	
	// Check for changes in sequence parameters
	var alterParts []string
	
	if d.Old.Increment != d.New.Increment {
		alterParts = append(alterParts, fmt.Sprintf("INCREMENT BY %d", d.New.Increment))
	}
	
	if (d.Old.MinValue == nil && d.New.MinValue != nil) || 
	   (d.Old.MinValue != nil && d.New.MinValue != nil && *d.Old.MinValue != *d.New.MinValue) {
		if d.New.MinValue != nil {
			alterParts = append(alterParts, fmt.Sprintf("MINVALUE %d", *d.New.MinValue))
		}
	}
	
	if (d.Old.MaxValue == nil && d.New.MaxValue != nil) || 
	   (d.Old.MaxValue != nil && d.New.MaxValue != nil && *d.Old.MaxValue != *d.New.MaxValue) {
		if d.New.MaxValue != nil {
			alterParts = append(alterParts, fmt.Sprintf("MAXVALUE %d", *d.New.MaxValue))
		}
	}
	
	if d.Old.StartValue != d.New.StartValue {
		alterParts = append(alterParts, fmt.Sprintf("RESTART WITH %d", d.New.StartValue))
	}
	
	// Cache is not part of ir.Sequence, skip it
	
	if d.Old.CycleOption != d.New.CycleOption {
		if d.New.CycleOption {
			alterParts = append(alterParts, "CYCLE")
		} else {
			alterParts = append(alterParts, "NO CYCLE")
		}
	}
	
	if len(alterParts) > 0 {
		statements = append(statements, fmt.Sprintf("ALTER SEQUENCE %s %s;", seqName, strings.Join(alterParts, " ")))
	}
	
	// Owner is tracked by OwnedByTable/OwnedByColumn, not directly
	
	return statements
}

// sequencesEqual checks if two sequences are structurally equal
func sequencesEqual(old, new *ir.Sequence) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Name != new.Name {
		return false
	}
	if old.StartValue != new.StartValue {
		return false
	}
	if old.Increment != new.Increment {
		return false
	}
	if (old.MinValue == nil && new.MinValue != nil) || 
	   (old.MinValue != nil && new.MinValue == nil) ||
	   (old.MinValue != nil && new.MinValue != nil && *old.MinValue != *new.MinValue) {
		return false
	}
	if (old.MaxValue == nil && new.MaxValue != nil) || 
	   (old.MaxValue != nil && new.MaxValue == nil) ||
	   (old.MaxValue != nil && new.MaxValue != nil && *old.MaxValue != *new.MaxValue) {
		return false
	}
	if old.CycleOption != new.CycleOption {
		return false
	}
	if old.OwnedByTable != new.OwnedByTable || old.OwnedByColumn != new.OwnedByColumn {
		return false
	}
	
	return true
}