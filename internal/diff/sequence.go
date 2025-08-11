package diff

import (
	"fmt"
	"math"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// Default values for PostgreSQL sequences by data type
const (
	defaultSequenceMinValue int64 = 1
	defaultSequenceMaxValue int64 = math.MaxInt64  // bigint max
	smallintMaxValue        int64 = math.MaxInt16   // smallint max  
	integerMaxValue         int64 = math.MaxInt32   // integer max
)

// generateCreateSequencesSQL generates CREATE SEQUENCE statements
func generateCreateSequencesSQL(sequences []*ir.Sequence, targetSchema string, collector *diffCollector) {
	for _, seq := range sequences {
		sql := generateSequenceSQL(seq, targetSchema)

		// Create context for this statement
		context := &diffContext{
			Type:                "sequence",
			Operation:           "create",
			Path:                fmt.Sprintf("%s.%s", seq.Schema, seq.Name),
			Source:              seq,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateDropSequencesSQL generates DROP SEQUENCE statements
func generateDropSequencesSQL(sequences []*ir.Sequence, targetSchema string, collector *diffCollector) {
	// Process sequences in reverse order (already sorted)
	for _, seq := range sequences {
		seqName := qualifyEntityName(seq.Schema, seq.Name, targetSchema)
		sql := fmt.Sprintf("DROP SEQUENCE IF EXISTS %s CASCADE;", seqName)

		// Create context for this statement
		context := &diffContext{
			Type:                "sequence",
			Operation:           "drop",
			Path:                fmt.Sprintf("%s.%s", seq.Schema, seq.Name),
			Source:              seq,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateModifySequencesSQL generates ALTER SEQUENCE statements
func generateModifySequencesSQL(diffs []*sequenceDiff, targetSchema string, collector *diffCollector) {
	for _, diff := range diffs {
		statements := diff.generateAlterSequenceStatements(targetSchema)
		for _, stmt := range statements {

			// Create context for this statement
			context := &diffContext{
				Type:                "sequence",
				Operation:           "alter",
				Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
				Source:              diff,
				CanRunInTransaction: true,
			}

			collector.collect(context, stmt)
		}
	}
}

// generateSequenceSQL generates CREATE SEQUENCE statement
func generateSequenceSQL(seq *ir.Sequence, targetSchema string) string {
	var parts []string

	seqName := qualifyEntityName(seq.Schema, seq.Name, targetSchema)
	parts = append(parts, fmt.Sprintf("CREATE SEQUENCE IF NOT EXISTS %s", seqName))

	// Add data type if specified (even if it's bigint, since user explicitly specified it)
	if seq.DataType != "" {
		parts = append(parts, fmt.Sprintf("AS %s", seq.DataType))
	}

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

	if seq.MaxValue != nil && *seq.MaxValue != defaultSequenceMaxValue {
		parts = append(parts, fmt.Sprintf("MAXVALUE %d", *seq.MaxValue))
	}

	// Add cache if it differs from default (1)
	if seq.Cache != nil && *seq.Cache != 1 {
		parts = append(parts, fmt.Sprintf("CACHE %d", *seq.Cache))
	}

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
func (d *sequenceDiff) generateAlterSequenceStatements(targetSchema string) []string {
	var statements []string

	seqName := qualifyEntityName(d.New.Schema, d.New.Name, targetSchema)

	// Check for changes in sequence parameters
	var alterParts []string

	if d.Old.Increment != d.New.Increment {
		alterParts = append(alterParts, fmt.Sprintf("INCREMENT BY %d", d.New.Increment))
	}

	// Handle MinValue changes with defaults
	oldMin := defaultSequenceMinValue
	if d.Old.MinValue != nil {
		oldMin = *d.Old.MinValue
	}
	newMin := defaultSequenceMinValue
	if d.New.MinValue != nil {
		newMin = *d.New.MinValue
	}
	if oldMin != newMin {
		alterParts = append(alterParts, fmt.Sprintf("MINVALUE %d", newMin))
	}

	// Handle MaxValue changes with defaults
	oldMax := defaultSequenceMaxValue
	if d.Old.MaxValue != nil {
		oldMax = *d.Old.MaxValue
	}
	newMax := defaultSequenceMaxValue
	if d.New.MaxValue != nil {
		newMax = *d.New.MaxValue
	}
	if oldMax != newMax {
		alterParts = append(alterParts, fmt.Sprintf("MAXVALUE %d", newMax))
	}

	if d.Old.StartValue != d.New.StartValue {
		alterParts = append(alterParts, fmt.Sprintf("RESTART WITH %d", d.New.StartValue))
	}

	// Handle Cache changes
	oldCache := int64(1)
	if d.Old.Cache != nil {
		oldCache = *d.Old.Cache
	}
	newCache := int64(1)
	if d.New.Cache != nil {
		newCache = *d.New.Cache
	}
	if oldCache != newCache {
		alterParts = append(alterParts, fmt.Sprintf("CACHE %d", newCache))
	}

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
	
	// Compare DataType (default is bigint if empty)
	oldDataType := old.DataType
	if oldDataType == "" {
		oldDataType = "bigint"
	}
	newDataType := new.DataType
	if newDataType == "" {
		newDataType = "bigint"
	}
	if oldDataType != newDataType {
		return false
	}
	
	if old.StartValue != new.StartValue {
		return false
	}
	if old.Increment != new.Increment {
		return false
	}

	// Handle MinValue comparison with defaults
	oldMin := defaultSequenceMinValue
	if old.MinValue != nil {
		oldMin = *old.MinValue
	}
	newMin := defaultSequenceMinValue
	if new.MinValue != nil {
		newMin = *new.MinValue
	}
	if oldMin != newMin {
		return false
	}

	// Handle MaxValue comparison with defaults
	oldMax := defaultSequenceMaxValue
	if old.MaxValue != nil {
		oldMax = *old.MaxValue
	}
	newMax := defaultSequenceMaxValue
	if new.MaxValue != nil {
		newMax = *new.MaxValue
	}
	if oldMax != newMax {
		return false
	}

	// Handle Cache comparison with defaults
	oldCache := int64(1)
	if old.Cache != nil {
		oldCache = *old.Cache
	}
	newCache := int64(1)
	if new.Cache != nil {
		newCache = *new.Cache
	}
	if oldCache != newCache {
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
