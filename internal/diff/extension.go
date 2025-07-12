package diff

import (
	"fmt"
	"sort"

	"github.com/pgschema/pgschema/internal/ir"
)

// generateCreateExtensionsSQL generates CREATE EXTENSION statements
func generateCreateExtensionsSQL(w *SQLWriter, extensions []*ir.Extension, targetSchema string) {
	// Sort extensions by name for consistent ordering
	sortedExtensions := make([]*ir.Extension, len(extensions))
	copy(sortedExtensions, extensions)
	sort.Slice(sortedExtensions, func(i, j int) bool {
		return sortedExtensions[i].Name < sortedExtensions[j].Name
	})

	for _, ext := range sortedExtensions {
		w.WriteDDLSeparator()
		sql := ""
		if ext.Schema != "" {
			sql = fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s WITH SCHEMA %s;", ext.Name, ext.Schema)
		} else {
			sql = fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s;", ext.Name)
		}
		w.WriteStatementWithComment("EXTENSION", ext.Name, ext.Schema, "", sql, targetSchema)
	}
}

// generateDropExtensionsSQL generates DROP EXTENSION statements
func generateDropExtensionsSQL(w *SQLWriter, extensions []*ir.Extension, targetSchema string) {
	// Sort extensions by name for consistent ordering
	sortedExtensions := make([]*ir.Extension, len(extensions))
	copy(sortedExtensions, extensions)
	sort.Slice(sortedExtensions, func(i, j int) bool {
		return sortedExtensions[i].Name < sortedExtensions[j].Name
	})

	for _, ext := range sortedExtensions {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP EXTENSION IF EXISTS %s;", ext.Name)
		w.WriteStatementWithComment("EXTENSION", ext.Name, ext.Schema, "", sql, targetSchema)
	}
}
