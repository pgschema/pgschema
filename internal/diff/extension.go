package diff

import (
	"fmt"
	"sort"

	"github.com/pgschema/pgschema/internal/ir"
)

// GenerateDropExtensionSQL generates SQL for dropping extensions
func GenerateDropExtensionSQL(extensions []*ir.Extension) []string {
	var statements []string
	
	// Sort extensions by name for consistent ordering
	sortedExtensions := make([]*ir.Extension, len(extensions))
	copy(sortedExtensions, extensions)
	sort.Slice(sortedExtensions, func(i, j int) bool {
		return sortedExtensions[i].Name < sortedExtensions[j].Name
	})
	
	for _, ext := range sortedExtensions {
		statements = append(statements, fmt.Sprintf("DROP EXTENSION IF EXISTS %s;", ext.Name))
	}
	
	return statements
}

// GenerateCreateExtensionSQL generates SQL for creating extensions
func GenerateCreateExtensionSQL(extensions []*ir.Extension) []string {
	var statements []string
	
	// Sort extensions by name for consistent ordering
	sortedExtensions := make([]*ir.Extension, len(extensions))
	copy(sortedExtensions, extensions)
	sort.Slice(sortedExtensions, func(i, j int) bool {
		return sortedExtensions[i].Name < sortedExtensions[j].Name
	})
	
	for _, ext := range sortedExtensions {
		if ext.Schema != "" {
			statements = append(statements, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s WITH SCHEMA %s;", ext.Name, ext.Schema))
		} else {
			statements = append(statements, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s;", ext.Name))
		}
	}
	
	return statements
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
		sql := generateExtensionSQL(ext)
		w.WriteStatementWithComment("EXTENSION", ext.Name, ext.Schema, "", sql, targetSchema)
	}
}

// generateExtensionSQL generates CREATE EXTENSION statement
func generateExtensionSQL(ext *ir.Extension) string {
	if ext.Schema != "" {
		return fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s WITH SCHEMA %s;", ext.Name, ext.Schema)
	} else {
		return fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s;", ext.Name)
	}
}