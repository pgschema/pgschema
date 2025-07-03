package ir

import (
	"fmt"
	"strings"
)

// Index represents a database index
type Index struct {
	Schema       string         `json:"schema"`
	Table        string         `json:"table"`
	Name         string         `json:"name"`
	Type         IndexType      `json:"type"`
	Method       string         `json:"method"` // btree, hash, gin, gist, etc.
	Columns      []*IndexColumn `json:"columns"`
	IsUnique     bool           `json:"is_unique"`
	IsPrimary    bool           `json:"is_primary"`
	IsPartial    bool           `json:"is_partial"`
	IsConcurrent bool           `json:"is_concurrent"`
	Where        string         `json:"where,omitempty"` // partial index condition
	Definition   string         `json:"definition"`      // full CREATE INDEX statement
	Comment      string         `json:"comment,omitempty"`
}

// IndexColumn represents a column within an index
type IndexColumn struct {
	Name      string `json:"name"`
	Position  int    `json:"position"`
	Direction string `json:"direction,omitempty"` // ASC, DESC
	Operator  string `json:"operator,omitempty"`  // operator class
}

// GenerateSQL for Index with target schema context
func (i *Index) GenerateSQL(targetSchema string) string {
	w := NewSQLWriter()

	var stmt string
	if i.Schema != targetSchema {
		// Use the definition as-is
		stmt = fmt.Sprintf("%s;", i.Definition)
	} else {
		// Remove schema qualifiers from the definition for schema-agnostic output
		definition := i.Definition
		schemaPrefix := i.Schema + "."
		// Remove schema qualifiers that match the target schema
		definition = strings.ReplaceAll(definition, schemaPrefix, "")
		stmt = fmt.Sprintf("%s;", definition)
	}

	// Remove "USING btree" since btree is the default index method
	if i.Method == "btree" {
		stmt = strings.ReplaceAll(stmt, " USING btree", "")
	}

	w.WriteStatementWithComment("INDEX", i.Name, i.Schema, "", stmt, targetSchema)
	return w.String()
}
