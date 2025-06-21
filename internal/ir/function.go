package ir

import (
	"fmt"
	"strings"
)

// Function represents a database function
type Function struct {
	Schema     string       `json:"schema"`
	Name       string       `json:"name"`
	Definition string       `json:"definition"`
	ReturnType string       `json:"return_type"`
	Language   string       `json:"language"`
	Arguments  string       `json:"arguments,omitempty"`
	Signature  string       `json:"signature,omitempty"`
	Parameters []*Parameter `json:"parameters,omitempty"`
	Comment           string       `json:"comment,omitempty"`
	Volatility        string       `json:"volatility,omitempty"`         // IMMUTABLE, STABLE, VOLATILE
	IsStrict          bool         `json:"is_strict,omitempty"`          // STRICT or null behavior
	IsSecurityDefiner bool         `json:"is_security_definer,omitempty"` // SECURITY DEFINER
}

// Parameter represents a function parameter
type Parameter struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Mode     string `json:"mode"` // IN, OUT, INOUT
	Position int    `json:"position"`
}

// GenerateSQL for Function
func (f *Function) GenerateSQL() string {
	if f.Definition == "<nil>" || f.Definition == "" {
		return ""
	}
	w := NewSQLWriter()
	
	// Build function signature for comment header (types only with schema qualifiers)
	headerSig := fmt.Sprintf("%s(%s)", f.Name, f.Arguments)
	
	// Build full function signature for CREATE statement (with parameter names)
	var createSig string
	if f.Signature != "" && f.Signature != "<nil>" {
		createSig = fmt.Sprintf("%s(%s)", f.Name, f.Signature)
	} else {
		createSig = fmt.Sprintf("%s(%s)", f.Name, f.Arguments)
	}
	
	// Build qualifiers (volatility and strictness)
	var qualifiers []string
	if f.Volatility != "" && f.Volatility != "VOLATILE" {
		// Only include non-default volatility (VOLATILE is the default, so omit it)
		qualifiers = append(qualifiers, f.Volatility)
	}
	if f.IsStrict {
		qualifiers = append(qualifiers, "STRICT")
	}
	if f.IsSecurityDefiner {
		qualifiers = append(qualifiers, "SECURITY DEFINER")
	}
	
	qualifierStr := ""
	if len(qualifiers) > 0 {
		qualifierStr = " " + strings.Join(qualifiers, " ")
	}
	
	// Generate CREATE FUNCTION statement
	stmt := fmt.Sprintf("CREATE FUNCTION %s.%s RETURNS %s\n    LANGUAGE %s%s\n    AS $$%s$$;",
		f.Schema, createSig, f.ReturnType, strings.ToLower(f.Language), qualifierStr, f.Definition)
	w.WriteStatementWithComment("FUNCTION", headerSig, f.Schema, "", stmt)
	
	// Generate COMMENT ON FUNCTION statement if comment exists
	if f.Comment != "" && f.Comment != "<nil>" {
		w.WriteDDLSeparator()
		
		// Escape single quotes in comment
		escapedComment := strings.ReplaceAll(f.Comment, "'", "''")
		commentStmt := fmt.Sprintf("COMMENT ON FUNCTION %s.%s IS '%s';", f.Schema, headerSig, escapedComment)
		w.WriteStatementWithComment("COMMENT", "FUNCTION "+headerSig, f.Schema, "", commentStmt)
	}
	
	return w.String()
}