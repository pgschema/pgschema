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