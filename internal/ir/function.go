package ir

// Function represents a database function
type Function struct {
	Schema            string       `json:"schema"`
	Name              string       `json:"name"`
	Definition        string       `json:"definition"`
	ReturnType        string       `json:"return_type"`
	Language          string       `json:"language"`
	Arguments         string       `json:"arguments,omitempty"`
	Signature         string       `json:"signature,omitempty"`
	Parameters        []*Parameter `json:"parameters,omitempty"`
	Comment           string       `json:"comment,omitempty"`
	Volatility        string       `json:"volatility,omitempty"`          // IMMUTABLE, STABLE, VOLATILE
	IsStrict          bool         `json:"is_strict,omitempty"`           // STRICT or null behavior
	IsSecurityDefiner bool         `json:"is_security_definer,omitempty"` // SECURITY DEFINER
}

// Parameter represents a function parameter
type Parameter struct {
	Name         string  `json:"name"`
	DataType     string  `json:"data_type"`
	Mode         string  `json:"mode"` // IN, OUT, INOUT
	Position     int     `json:"position"`
	DefaultValue *string `json:"default_value,omitempty"`
}
