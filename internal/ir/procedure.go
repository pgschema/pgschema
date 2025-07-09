package ir

import (
)

// Procedure represents a database procedure
type Procedure struct {
	Schema     string       `json:"schema"`
	Name       string       `json:"name"`
	Definition string       `json:"definition"`
	Language   string       `json:"language"`
	Arguments  string       `json:"arguments,omitempty"`
	Signature  string       `json:"signature,omitempty"`
	Parameters []*Parameter `json:"parameters,omitempty"`
	Comment    string       `json:"comment,omitempty"`
}


