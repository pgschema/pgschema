package dump

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/version"
)

// DumpFormatter handles formatting SQL output for database dumps
type DumpFormatter struct {
	dbVersion    string
	targetSchema string
}

// NewDumpFormatter creates a new DumpFormatter
func NewDumpFormatter(dbVersion string, targetSchema string) *DumpFormatter {
	return &DumpFormatter{
		dbVersion:    dbVersion,
		targetSchema: targetSchema,
	}
}

// FormatSingleFile formats SQL output for single-file dump with pg_dump-style headers
func (f *DumpFormatter) FormatSingleFile(diffs []diff.Diff) string {
	var output strings.Builder

	// Generate and write header
	header := f.generateDumpHeader()
	output.WriteString(header)

	// Format SQL with pg_dump-style formatting
	for i, step := range diffs {
		if strings.ToUpper(step.Type) == "COMMENT" {
			// For comments, just write the raw SQL without DDL header
			if i > 0 {
				output.WriteString("\n") // Add separator from previous statement
			}
			output.WriteString(step.SQL)
			if !strings.HasSuffix(step.SQL, "\n") {
				output.WriteString("\n")
			}
		} else {
			// Add DDL separator with comment header for non-comment statements
			output.WriteString("--\n")

			// Determine schema name for comment (use "-" for target schema)
			commentSchemaName := step.Path
			if strings.Contains(step.Path, ".") {
				parts := strings.Split(step.Path, ".")
				if len(parts) >= 2 && parts[0] == f.targetSchema {
					commentSchemaName = "-"
				} else {
					commentSchemaName = parts[0]
				}
			}

			// Print object comment header
			objectName := step.Path
			if strings.Contains(step.Path, ".") {
				parts := strings.Split(step.Path, ".")
				if len(parts) >= 2 {
					objectName = parts[1]
				}
			}

			output.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: -\n", objectName, strings.ToUpper(step.Type), commentSchemaName))
			output.WriteString("--\n")
			output.WriteString("\n")

			// Add the SQL statement
			output.WriteString(step.SQL)
		}

		// Add newline after SQL, and extra newline only if not last item
		if i < len(diffs)-1 {
			output.WriteString("\n\n")
		}
	}

	return output.String()
}

// FormatMultiFile creates multiple SQL files organized by object type
func (f *DumpFormatter) FormatMultiFile(diffs []diff.Diff, outputPath string) error {
	// Create base directory
	baseDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Organization by object type
	filesByType := make(map[string]map[string][]diff.Diff)
	includes := []string{}

	// Group diffs by object type and name
	for _, step := range diffs {
		objType := strings.ToUpper(step.Type)

		// Determine directory and object name
		var dir string
		if objType == "COMMENT" {
			// Special handling for comments - use parent directory
			dir = f.getCommentParentDirectory(step)
		} else {
			dir = f.getObjectDirectory(objType)
		}
		objName := f.getGroupingName(step)

		if filesByType[dir] == nil {
			filesByType[dir] = make(map[string][]diff.Diff)
		}

		filesByType[dir][objName] = append(filesByType[dir][objName], step)
	}

	// Create files in dependency order
	orderedDirs := []string{"types", "domains", "sequences", "functions", "procedures", "tables", "views"}

	for _, dir := range orderedDirs {
		if objects, exists := filesByType[dir]; exists {
			// Create directory
			dirPath := filepath.Join(baseDir, dir)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
			}

			// Create files for each object
			for objName, objSteps := range objects {
				fileName := f.sanitizeFileName(objName) + ".sql"
				filePath := filepath.Join(dirPath, fileName)
				relativePath := filepath.Join(dir, fileName)

				// Write object file
				if err := f.writeObjectFile(filePath, objSteps); err != nil {
					return fmt.Errorf("failed to write file %s: %w", filePath, err)
				}

				// Add include statement
				includes = append(includes, fmt.Sprintf("\\i %s", relativePath))
			}
		}
	}

	// Create main file with header and includes
	mainFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create main file: %w", err)
	}
	defer mainFile.Close()

	// Write header
	header := f.generateDumpHeader()
	mainFile.WriteString(header)

	// Write includes
	for _, include := range includes {
		mainFile.WriteString(include + "\n")
	}

	// Remove trailing newline (to match original behavior)
	if len(includes) > 0 {
		pos, _ := mainFile.Seek(-1, 2) // Go to last character
		var lastChar [1]byte
		mainFile.Read(lastChar[:])
		if lastChar[0] == '\n' {
			mainFile.Truncate(pos) // Remove last newline
		}
	}

	return nil
}

// generateDumpHeader generates the header for database dumps with metadata
func (f *DumpFormatter) generateDumpHeader() string {
	var header strings.Builder

	header.WriteString("--\n")
	header.WriteString("-- pgschema database dump\n")
	header.WriteString("--\n")
	header.WriteString("\n")

	header.WriteString(fmt.Sprintf("-- Dumped from database version %s\n", f.dbVersion))
	header.WriteString(fmt.Sprintf("-- Dumped by pgschema version %s\n", version.App()))
	header.WriteString("\n")
	header.WriteString("\n")
	return header.String()
}

// writeObjectFile writes a single object file with its SQL statements
func (f *DumpFormatter) writeObjectFile(filePath string, diffs []diff.Diff) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for i, step := range diffs {
		isComment := strings.ToUpper(step.Type) == "COMMENT"

		if isComment {
			// For comments, add a blank line before the comment
			file.WriteString("\n")
			file.WriteString(step.SQL)
			if !strings.HasSuffix(step.SQL, "\n") {
				file.WriteString("\n")
			}
		} else {
			// For non-comment statements, check if we need spacing
			if i > 0 {
				// Check if previous step was a comment
				prevIsComment := strings.ToUpper(diffs[i-1].Type) == "COMMENT"
				if prevIsComment {
					// Add extra newline after comment before next object
					file.WriteString("\n")
				} else {
					// Normal spacing between non-comment objects - just one newline
					file.WriteString("\n")
				}
			}

			// Add DDL separator with comment header for non-comment statements
			file.WriteString("--\n")

			// Determine schema name for comment (use "-" for target schema)
			commentSchemaName := step.Path
			if strings.Contains(step.Path, ".") {
				parts := strings.Split(step.Path, ".")
				if len(parts) >= 2 && parts[0] == f.targetSchema {
					commentSchemaName = "-"
				} else {
					commentSchemaName = parts[0]
				}
			}

			// Print object comment header
			objectName := step.Path
			if strings.Contains(step.Path, ".") {
				parts := strings.Split(step.Path, ".")
				if len(parts) >= 2 {
					objectName = parts[1]
				}
			}

			file.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: -\n", objectName, strings.ToUpper(step.Type), commentSchemaName))
			file.WriteString("--\n")
			file.WriteString("\n")

			// Print the SQL statement
			file.WriteString(step.SQL)
			// Ensure non-comment statements end with a newline
			if !strings.HasSuffix(step.SQL, "\n") {
				file.WriteString("\n")
			}
		}
	}

	// Trim trailing newlines from the file (similar to MultiFileWriter behavior)
	file.Close()

	// Read the file content to trim trailing newlines
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Trim trailing newlines and rewrite
	trimmedContent := strings.TrimRight(string(content), "\n")
	return os.WriteFile(filePath, []byte(trimmedContent), 0644)
}

// getObjectDirectory returns the directory name for an object type
func (f *DumpFormatter) getObjectDirectory(objectType string) string {
	switch objectType {
	case "TYPE":
		return "types"
	case "DOMAIN":
		return "domains"
	case "SEQUENCE":
		return "sequences"
	case "FUNCTION":
		return "functions"
	case "PROCEDURE":
		return "procedures"
	case "TABLE":
		return "tables"
	case "VIEW", "MATERIALIZED VIEW":
		return "views"
	case "TRIGGER", "INDEX", "CONSTRAINT", "POLICY", "RULE":
		// These are included with their tables
		return "tables"
	case "COMMENT":
		// Comments handled separately in FormatMultiFile
		return "tables" // fallback, will be overridden
	default:
		return "misc"
	}
}

// getGroupingName determines the appropriate name for grouping objects into files
func (f *DumpFormatter) getGroupingName(step diff.Diff) string {
	objType := strings.ToUpper(step.Type)

	// For table-related objects, try to extract the table name from Source
	switch objType {
	case "INDEX", "CONSTRAINT", "POLICY", "TRIGGER", "RULE":
		if tableName := f.extractTableNameFromContext(step); tableName != "" {
			return tableName
		}
	case "COMMENT":
		// For comments, we need to determine the parent object
		// For index comments, group with parent table
		if step.Source != nil {
			switch obj := step.Source.(type) {
			case *ir.Index:
				return obj.Table // Group index comments with their table
			case *ir.Table:
				return obj.Name // Table and column comments group with table
			case *ir.View:
				return obj.Name // View comments group with view
			}
		}
		// Fallback: extract parent object name from object path
		// Path format: "schema.table" or "schema.table.column" for table/column comments
		// Path format: "schema.view" for view comments
		if parts := strings.Split(step.Path, "."); len(parts) >= 2 {
			return parts[1] // Return parent object name (table/view)
		}
	}

	// For standalone objects or if table name extraction fails, use object name
	return f.getObjectName(step.Path)
}

// extractTableNameFromContext extracts table name from the Source in diffContext
func (f *DumpFormatter) extractTableNameFromContext(step diff.Diff) string {
	if step.Source == nil {
		return ""
	}

	// Try to extract table name based on the type of Source
	switch obj := step.Source.(type) {
	case *ir.Index:
		return obj.Table
	case *ir.RLSPolicy:
		return obj.Table
	case *ir.Trigger:
		return obj.Table
	// For other table-related objects, we might need to parse or handle differently
	default:
		return ""
	}
}

// getCommentParentDirectory determines the directory for comment statements based on their parent object
func (f *DumpFormatter) getCommentParentDirectory(step diff.Diff) string {
	if step.Source != nil {
		switch step.Source.(type) {
		case *ir.View:
			return "views"
		case *ir.Table, *ir.Index:
			return "tables"
		}
	}
	// Fallback to tables for unknown comment types
	return "tables"
}

// getObjectName extracts the object name from the object path
func (f *DumpFormatter) getObjectName(objectPath string) string {
	if strings.Contains(objectPath, ".") {
		parts := strings.Split(objectPath, ".")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return objectPath
}

// sanitizeFileName converts an object name to a valid filename
func (f *DumpFormatter) sanitizeFileName(name string) string {
	// Replace non-alphanumeric characters with underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	sanitized := reg.ReplaceAllString(name, "_")

	// Remove leading/trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	// Convert to lowercase for consistency
	return strings.ToLower(sanitized)
}
