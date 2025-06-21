package ir

import (
	"fmt"
	"strings"
)

// Aggregate represents a database aggregate function
type Aggregate struct {
	Schema                   string `json:"schema"`
	Name                     string `json:"name"`
	Arguments                string `json:"arguments,omitempty"`
	Signature                string `json:"signature,omitempty"`
	ReturnType               string `json:"return_type"`
	TransitionFunction       string `json:"transition_function"`
	TransitionFunctionSchema string `json:"transition_function_schema,omitempty"`
	StateType                string `json:"state_type"`
	InitialCondition         string `json:"initial_condition,omitempty"`
	FinalFunction            string `json:"final_function,omitempty"`
	FinalFunctionSchema      string `json:"final_function_schema,omitempty"`
	Comment                  string `json:"comment,omitempty"`
}

// GenerateSQL for Aggregate
func (a *Aggregate) GenerateSQL() string {
	if a.Name == "" || a.TransitionFunction == "" || a.StateType == "" {
		return ""
	}
	w := NewSQLWriter()
	
	// Build aggregate signature for comment header
	headerSig := fmt.Sprintf("%s(%s)", a.Name, a.Arguments)
	
	// Build full aggregate signature for CREATE statement
	var createSig string
	if a.Signature != "" && a.Signature != "<nil>" {
		createSig = fmt.Sprintf("%s(%s)", a.Name, a.Signature)
	} else {
		createSig = fmt.Sprintf("%s(%s)", a.Name, a.Arguments)
	}
	
	// Build the CREATE AGGREGATE statement
	var parts []string
	
	// Start with CREATE AGGREGATE
	createStmt := fmt.Sprintf("CREATE AGGREGATE %s.%s (", a.Schema, createSig)
	parts = append(parts, createStmt)
	
	// Add SFUNC (state function)
	sfuncName := a.TransitionFunction
	if a.TransitionFunctionSchema != "" && a.TransitionFunctionSchema != "<nil>" && a.TransitionFunctionSchema != a.Schema {
		sfuncName = fmt.Sprintf("%s.%s", a.TransitionFunctionSchema, a.TransitionFunction)
	}
	parts = append(parts, fmt.Sprintf("    SFUNC = %s,", sfuncName))
	
	// Add STYPE (state type)
	parts = append(parts, fmt.Sprintf("    STYPE = %s", a.StateType))
	
	// Add INITCOND if present (even if empty string, as it's valid)
	// Only skip if it's explicitly null/nil
	if a.InitialCondition != "<nil>" && a.InitialCondition != "NULL" && a.InitialCondition != "null" {
		// Remove the comma from STYPE and add it before INITCOND
		if len(parts) > 0 {
			lastIdx := len(parts) - 1
			if !strings.HasSuffix(parts[lastIdx], ",") {
				parts[lastIdx] = parts[lastIdx] + ","
			}
		}
		parts = append(parts, fmt.Sprintf("    INITCOND = '%s'", a.InitialCondition))
	}
	
	// Add FINALFUNC if present
	if a.FinalFunction != "" && a.FinalFunction != "<nil>" {
		ffuncName := a.FinalFunction
		if a.FinalFunctionSchema != "" && a.FinalFunctionSchema != "<nil>" && a.FinalFunctionSchema != a.Schema {
			ffuncName = fmt.Sprintf("%s.%s", a.FinalFunctionSchema, a.FinalFunction)
		}
		// Remove the comma from STYPE and add it before FINALFUNC
		if len(parts) > 0 {
			lastIdx := len(parts) - 1
			if !strings.HasSuffix(parts[lastIdx], ",") {
				parts[lastIdx] = parts[lastIdx] + ","
			}
		}
		parts = append(parts, fmt.Sprintf("    FINALFUNC = %s", ffuncName))
	}
	
	// Close the statement
	parts = append(parts, ");")
	
	stmt := strings.Join(parts, "\n")
	w.WriteStatementWithComment("AGGREGATE", headerSig, a.Schema, "", stmt)
	
	// Generate COMMENT ON AGGREGATE statement if comment exists
	if a.Comment != "" && a.Comment != "<nil>" {
		w.WriteDDLSeparator()
		
		// Escape single quotes in comment
		escapedComment := strings.ReplaceAll(a.Comment, "'", "''")
		commentStmt := fmt.Sprintf("COMMENT ON AGGREGATE %s.%s IS '%s';", a.Schema, headerSig, escapedComment)
		w.WriteStatementWithComment("COMMENT", "AGGREGATE "+headerSig, a.Schema, "", commentStmt)
	}
	
	return w.String()
}