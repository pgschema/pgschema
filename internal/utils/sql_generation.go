package utils

import (
	"fmt"
	"strings"
)

// Entity represents any database entity that can generate SQL
type Entity interface {
	GetSchema() string
	GetName() string
	GetComment() string
}

// SQLGenerationOptions contains options for SQL generation
type SQLGenerationOptions struct {
	IncludeComments bool
	TargetSchema    string
	EntityType      string
}

// SQLWriter interface to abstract the writer
type SQLWriter interface {
	WriteString(s string)
	WriteStatementWithComment(entityType, name, schema, owner, stmt, targetSchema string)
	WriteDDLSeparator()
	String() string
}

// GenerateEntitySQL provides common SQL generation logic for entities
func GenerateEntitySQL(entity Entity, opts SQLGenerationOptions, stmt string, writer SQLWriter) string {
	// Get qualified entity name
	entityName := QualifyEntityName(entity.GetSchema(), entity.GetName(), opts.TargetSchema)
	
	// Replace placeholder in statement
	stmt = strings.ReplaceAll(stmt, "{ENTITY_NAME}", entityName)
	
	// Get comment schema name
	commentSchema := GetCommentSchemaName(entity.GetSchema(), opts.TargetSchema)
	
	// Write statement with or without comments
	if opts.IncludeComments {
		writer.WriteStatementWithComment(opts.EntityType, entity.GetName(), commentSchema, "", stmt, "")
	} else {
		writer.WriteString(stmt)
	}
	
	// Add entity comment if present
	if entity.GetComment() != "" && entity.GetComment() != "<nil>" && opts.IncludeComments {
		writer.WriteDDLSeparator()
		entityName := QualifyEntityName(entity.GetSchema(), entity.GetName(), opts.TargetSchema)
		escapedComment := strings.ReplaceAll(entity.GetComment(), "'", "''")
		commentStmt := fmt.Sprintf("COMMENT ON %s %s IS '%s';", 
			strings.ToUpper(opts.EntityType), entityName, escapedComment)
		writer.WriteStatementWithComment("COMMENT", strings.ToUpper(opts.EntityType)+" "+entity.GetName(), 
			commentSchema, "", commentStmt, "")
	}
	
	return writer.String()
}

// BuildStatementWithQualification creates a statement with proper entity name qualification
func BuildStatementWithQualification(template string, entity Entity, targetSchema string) string {
	entityName := QualifyEntityName(entity.GetSchema(), entity.GetName(), targetSchema)
	return strings.ReplaceAll(template, "{ENTITY_NAME}", entityName)
}

// ShouldQualifyEntity returns true if entity should be schema-qualified
func ShouldQualifyEntity(entitySchema, targetSchema string) bool {
	return ShouldIncludeSchema(entitySchema, targetSchema)
}