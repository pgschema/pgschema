package ir

import (
	"fmt"
)

// Index represents a database index
type Index struct {
	Schema     string         `json:"schema"`
	Table      string         `json:"table"`
	Name       string         `json:"name"`
	Type       IndexType      `json:"type"`
	Method     string         `json:"method"` // btree, hash, gin, gist, etc.
	Columns    []*IndexColumn `json:"columns"`
	IsUnique   bool           `json:"is_unique"`
	IsPrimary  bool           `json:"is_primary"`
	IsPartial  bool           `json:"is_partial"`
	Where      string         `json:"where,omitempty"` // partial index condition
	Definition string         `json:"definition"`      // full CREATE INDEX statement
	Comment    string         `json:"comment,omitempty"`
}

// IndexColumn represents a column within an index
type IndexColumn struct {
	Name      string `json:"name"`
	Position  int    `json:"position"`
	Direction string `json:"direction,omitempty"` // ASC, DESC
	Operator  string `json:"operator,omitempty"`  // operator class
}

// GenerateSQL for Index
func (i *Index) GenerateSQL() string {
	w := NewSQLWriter()
	stmt := fmt.Sprintf("%s;", i.Definition)
	w.WriteStatementWithComment("INDEX", i.Name, i.Schema, "", stmt)
	return w.String()
}