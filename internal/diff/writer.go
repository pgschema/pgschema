package diff

// Writer is an interface for writing SQL statements
type Writer interface {
	// WriteString writes a string to the output
	WriteString(s string)
	
	// WriteDDLSeparator writes DDL separator (newlines)
	WriteDDLSeparator()
	
	// WriteStatementWithComment writes a SQL statement with optional comment header
	WriteStatementWithComment(objectType, objectName, schemaName, owner string, stmt string, targetSchema string)
	
	// String returns the accumulated output
	String() string
}