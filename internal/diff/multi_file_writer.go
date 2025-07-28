package diff

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MultiFileWriter writes SQL statements to multiple files organized by object type
type MultiFileWriter struct {
	baseDir         string
	mainFile        *os.File
	includeComments bool
	includes        []string
	currentFile     *os.File
	currentOutput   strings.Builder
	currentFileBuffer strings.Builder
}

// NewMultiFileWriter creates a new MultiFileWriter
func NewMultiFileWriter(outputPath string, includeComments bool) (*MultiFileWriter, error) {
	// Extract base directory
	baseDir := filepath.Dir(outputPath)

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create main file
	mainFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create main file: %w", err)
	}

	// Note: Header will be written separately by the dump command

	return &MultiFileWriter{
		baseDir:         baseDir,
		mainFile:        mainFile,
		includeComments: includeComments,
		includes:        []string{},
	}, nil
}

// WriteHeader writes the header to the main file
func (w *MultiFileWriter) WriteHeader(header string) {
	w.mainFile.WriteString(header)
}

// WriteString writes a string to the current output
func (w *MultiFileWriter) WriteString(s string) {
	if w.currentFile != nil {
		w.currentFileBuffer.WriteString(s)
	} else {
		w.currentOutput.WriteString(s)
	}
}

// WriteDDLSeparator writes DDL separator
func (w *MultiFileWriter) WriteDDLSeparator() {
	w.WriteString("\n")
}

// closeCurrentFile writes the buffered content to the current file with proper newline handling
func (w *MultiFileWriter) closeCurrentFile() {
	if w.currentFile != nil && w.currentFileBuffer.Len() > 0 {
		// Trim trailing newlines from buffered content - NO trailing newlines like SingleFileWriter
		content := strings.TrimRight(w.currentFileBuffer.String(), "\n")
		if content != "" {
			w.currentFile.WriteString(content)
		}
		w.currentFileBuffer.Reset()
	}
	if w.currentFile != nil {
		w.currentFile.Close()
		w.currentFile = nil
	}
}

// WriteStatementWithComment writes a SQL statement with optional comment header
func (w *MultiFileWriter) WriteStatementWithComment(objectType, objectName, schemaName, owner string, stmt string, targetSchema string) {
	// Determine the file path for this object
	relPath := w.getObjectFilePath(objectType, objectName)

	// Check if we need to switch files
	needNewFile := false
	switch strings.ToUpper(objectType) {
	case "TYPE", "DOMAIN", "SEQUENCE", "FUNCTION", "PROCEDURE", "TABLE", "VIEW", "MATERIALIZED VIEW":
		needNewFile = true
	}

	if needNewFile {
		// Close current file if any (with proper newline handling)
		if w.currentFile != nil {
			w.closeCurrentFile()
		}

		// Ensure directory exists
		if err := w.ensureDirectory(relPath); err != nil {
			// If we can't create directory, write to main file
			w.writeToMainFile(objectType, objectName, schemaName, owner, stmt, targetSchema)
			return
		}

		// Create or append to the file
		fullPath := filepath.Join(w.baseDir, relPath)
		file, err := os.Create(fullPath)
		if err != nil {
			// If we can't create file, write to main file
			w.writeToMainFile(objectType, objectName, schemaName, owner, stmt, targetSchema)
			return
		}

		w.currentFile = file
		w.currentFileBuffer.Reset() // Reset buffer for new file

		// Add include directive to main file (only once per file)
		includeStmt := fmt.Sprintf("\\i %s", relPath)
		if !w.hasInclude(includeStmt) {
			w.includes = append(w.includes, includeStmt)
		}
	}

	// Write the statement with optional comment
	if w.includeComments {
		w.WriteString("--\n")
		commentSchemaName := schemaName
		if targetSchema != "" && schemaName == targetSchema {
			commentSchemaName = "-"
		}
		if owner != "" {
			w.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: %s\n", objectName, objectType, commentSchemaName, owner))
		} else {
			w.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: -\n", objectName, objectType, commentSchemaName))
		}
		w.WriteString("--\n\n")
	}

	w.WriteString(stmt)
	w.WriteString("\n")
}

// String returns the accumulated output and finalizes the main file
func (w *MultiFileWriter) String() string {
	// Close current file if any (with proper newline handling)
	if w.currentFile != nil {
		w.closeCurrentFile()
	}

	// Build main file content with includes and remaining content
	var mainContent strings.Builder
	
	// Write includes to main file content in dependency order
	// Order: types, domains, sequences, functions, procedures, tables, views
	orderedDirs := []string{"types", "domains", "sequences", "functions", "procedures", "tables", "views"}

	hasIncludes := false
	for _, dir := range orderedDirs {
		for _, include := range w.includes {
			if strings.HasPrefix(include, fmt.Sprintf("\\i %s/", dir)) {
				mainContent.WriteString(include + "\n")
				hasIncludes = true
			}
		}
	}

	// Write any remaining output that wasn't written to files
	if w.currentOutput.Len() > 0 {
		if hasIncludes {
			mainContent.WriteString("\n") // Add separator between includes and remaining content
		}
		remainingContent := strings.TrimRight(w.currentOutput.String(), "\n")
		if remainingContent != "" {
			mainContent.WriteString(remainingContent)
		}
	}

	// Write the final main content with proper trimming (like SingleFileWriter)
	finalContent := strings.TrimRight(mainContent.String(), "\n")
	if finalContent != "" {
		w.mainFile.WriteString(finalContent)
	}

	// Close main file
	w.mainFile.Close()

	// Return empty string since we don't want to emit the completion message
	return ""
}

// writeToMainFile writes directly to the main file (fallback)
func (w *MultiFileWriter) writeToMainFile(objectType, objectName, schemaName, owner string, stmt string, targetSchema string) {
	if w.includeComments {
		w.currentOutput.WriteString("--\n")
		commentSchemaName := schemaName
		if targetSchema != "" && schemaName == targetSchema {
			commentSchemaName = "-"
		}
		if owner != "" {
			w.currentOutput.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: %s\n", objectName, objectType, commentSchemaName, owner))
		} else {
			w.currentOutput.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: -\n", objectName, objectType, commentSchemaName))
		}
		w.currentOutput.WriteString("--\n\n")
	}

	w.currentOutput.WriteString(stmt)
	w.currentOutput.WriteString("\n")
}

// hasInclude checks if an include statement already exists
func (w *MultiFileWriter) hasInclude(include string) bool {
	for _, existing := range w.includes {
		if existing == include {
			return true
		}
	}
	return false
}

// sanitizeFileName converts an object name to a valid filename
func sanitizeFileName(name string) string {
	// Replace non-alphanumeric characters with underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	sanitized := reg.ReplaceAllString(name, "_")

	// Remove leading/trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	// Convert to lowercase for consistency
	return strings.ToLower(sanitized)
}

// getObjectFilePath returns the file path for a given object type and name
func (w *MultiFileWriter) getObjectFilePath(objectType, objectName string) string {
	var dir string
	switch strings.ToUpper(objectType) {
	case "TYPE":
		dir = "types"
	case "DOMAIN":
		dir = "domains"
	case "SEQUENCE":
		dir = "sequences"
	case "FUNCTION":
		dir = "functions"
	case "PROCEDURE":
		dir = "procedures"
	case "TABLE":
		dir = "tables"
	case "VIEW", "MATERIALIZED VIEW":
		dir = "views"
	case "TRIGGER":
		// Triggers are included with their tables
		dir = "tables"
	case "INDEX", "CONSTRAINT", "POLICY", "RULE", "COMMENT ON TABLE", "COMMENT ON COLUMN":
		// These are included with their tables
		dir = "tables"
	default:
		// Default to a misc directory for unknown types
		dir = "misc"
	}

	fileName := sanitizeFileName(objectName) + ".sql"
	return filepath.Join(dir, fileName)
}

// ensureDirectory creates the directory for the given path if it doesn't exist
func (w *MultiFileWriter) ensureDirectory(relPath string) error {
	fullPath := filepath.Join(w.baseDir, filepath.Dir(relPath))
	return os.MkdirAll(fullPath, 0755)
}
