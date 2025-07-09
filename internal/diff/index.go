package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// GenerateDropIndexSQL generates SQL for dropping indexes
func GenerateDropIndexSQL(indexes []*ir.Index) []string {
	var statements []string
	
	// Sort indexes by schema.table.name for consistent ordering
	sortedIndexes := make([]*ir.Index, len(indexes))
	copy(sortedIndexes, indexes)
	sort.Slice(sortedIndexes, func(i, j int) bool {
		keyI := sortedIndexes[i].Schema + "." + sortedIndexes[i].Table + "." + sortedIndexes[i].Name
		keyJ := sortedIndexes[j].Schema + "." + sortedIndexes[j].Table + "." + sortedIndexes[j].Name
		return keyI < keyJ
	})
	
	for _, index := range sortedIndexes {
		statements = append(statements, fmt.Sprintf("DROP INDEX IF EXISTS %s.%s;", index.Schema, index.Name))
	}
	
	return statements
}

// GenerateCreateIndexSQL generates SQL for creating indexes
func (d *DDLDiff) GenerateCreateIndexSQL(indexes []*ir.Index) []string {
	var statements []string
	
	// Sort indexes by schema.table.name for consistent ordering
	sortedIndexes := make([]*ir.Index, len(indexes))
	copy(sortedIndexes, indexes)
	sort.Slice(sortedIndexes, func(i, j int) bool {
		keyI := sortedIndexes[i].Schema + "." + sortedIndexes[i].Table + "." + sortedIndexes[i].Name
		keyJ := sortedIndexes[j].Schema + "." + sortedIndexes[j].Table + "." + sortedIndexes[j].Name
		return keyI < keyJ
	})
	
	for _, index := range sortedIndexes {
		// Generate clean migration SQL without schema qualifiers and USING btree
		indexSQL := d.generateIndexSQL(index, "")
		// Remove any comment headers and trailing newlines
		indexSQL = strings.TrimSpace(indexSQL)
		// Extract just the CREATE INDEX statement
		lines := strings.Split(indexSQL, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "CREATE") && strings.Contains(line, "INDEX") {
				statements = append(statements, line)
				break
			}
		}
	}
	
	return statements
}