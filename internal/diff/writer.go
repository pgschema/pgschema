package diff

// SQLContext provides context about the SQL statement being generated
type SQLContext struct {
	ObjectType   string // e.g., "table", "view", "function"
	Operation    string // e.g., "create", "alter", "drop"
	ObjectPath   string // e.g., "schema.table" or "schema.table.column"
	SourceChange any    // The DDLDiff element that generated this SQL
}

// PlanStep represents a single SQL statement with its source change
type PlanStep struct {
	SQL          string `json:"sql"`
	ObjectType   string `json:"object_type"`
	Operation    string `json:"operation"` // create, alter, drop
	ObjectPath   string `json:"object_path"`
	SourceChange any    `json:"source_change"`
}

// Writer is an interface for writing SQL statements
type Writer interface {
	// WriteString writes a string to the output
	WriteString(s string)
	
	// WriteDDLSeparator writes DDL separator (newlines)
	WriteDDLSeparator()
	
	// WriteStatementWithComment writes a SQL statement with optional comment header
	WriteStatementWithComment(objectType, objectName, schemaName, owner string, stmt string, targetSchema string)
	
	// WriteStatementWithContext writes a SQL statement with context and optional comment header
	WriteStatementWithContext(objectType, objectName, schemaName, owner string, stmt string, targetSchema string, context *SQLContext)
	
	// String returns the accumulated output
	String() string
}