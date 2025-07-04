package utils

import "fmt"

// QualifyEntityName returns the properly qualified entity name based on target schema
// If entity is in target schema, returns just the name, otherwise returns schema.name
func QualifyEntityName(entitySchema, entityName, targetSchema string) string {
	if entitySchema == targetSchema {
		return entityName
	}
	return fmt.Sprintf("%s.%s", entitySchema, entityName)
}

// GetCommentSchemaName returns the schema name for comments
// Returns "-" if entity is in target schema, otherwise returns the actual schema
func GetCommentSchemaName(entitySchema, targetSchema string) string {
	if entitySchema == targetSchema {
		return "-"
	}
	return entitySchema
}

// ShouldIncludeSchema returns true if we should include schema prefix for cross-schema references
func ShouldIncludeSchema(entitySchema, targetSchema string) bool {
	return entitySchema != targetSchema
}