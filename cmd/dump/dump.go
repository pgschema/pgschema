package dump

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pgschema/pgschema/cmd/util"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/version"
	"github.com/spf13/cobra"
)

var (
	host      string
	port      int
	db        string
	user      string
	password  string
	schema    string
	multiFile bool
	file      string
)

var DumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump database schema for a specific schema",
	Long:  "Dump and output database schema information for a specific schema. Uses the --schema flag to target a particular schema (defaults to 'public').",
	RunE:  runDump,
}

func init() {
	DumpCmd.Flags().StringVar(&host, "host", "localhost", "Database server host")
	DumpCmd.Flags().IntVar(&port, "port", 5432, "Database server port")
	DumpCmd.Flags().StringVar(&db, "db", "", "Database name (required)")
	DumpCmd.Flags().StringVar(&user, "user", "", "Database user name (required)")
	DumpCmd.Flags().StringVar(&password, "password", "", "Database password (optional, can also use PGPASSWORD env var)")
	DumpCmd.Flags().StringVar(&schema, "schema", "public", "Schema name to dump (default: public)")
	DumpCmd.Flags().BoolVar(&multiFile, "multi-file", false, "Output schema to multiple files organized by object type")
	DumpCmd.Flags().StringVar(&file, "file", "", "Output file path (required when --multi-file is used)")
	DumpCmd.MarkFlagRequired("db")
	DumpCmd.MarkFlagRequired("user")
}

// generateDumpHeader generates the header for database dumps with metadata
func generateDumpHeader(schemaIR *ir.IR) string {
	var header strings.Builder

	header.WriteString("--\n")
	header.WriteString("-- pgschema database dump\n")
	header.WriteString("--\n")
	header.WriteString("\n")

	header.WriteString(fmt.Sprintf("-- Dumped from database version %s\n", schemaIR.Metadata.DatabaseVersion))
	header.WriteString(fmt.Sprintf("-- Dumped by pgschema version %s\n", version.App()))
	header.WriteString("\n")
	header.WriteString("\n")
	return header.String()
}

// createMultiFileOutput creates multiple SQL files organized by object type
func createMultiFileOutput(collector *diff.SQLCollector, schemaIR *ir.IR, targetSchema, outputPath string) error {
	// Create base directory
	baseDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Organization by object type
	filesByType := make(map[string]map[string][]diff.PlanStep)
	includes := []string{}

	// Group steps by object type and name
	steps := collector.GetSteps()
	for _, step := range steps {
		objType := strings.ToUpper(step.ObjectType)

		// Determine directory and object name
		var dir string
		if objType == "COMMENT" {
			// Special handling for comments - use parent directory
			dir = getCommentParentDirectory(step)
		} else {
			dir = getObjectDirectory(objType)
		}
		objName := getGroupingName(step, targetSchema)

		if filesByType[dir] == nil {
			filesByType[dir] = make(map[string][]diff.PlanStep)
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
				fileName := sanitizeFileName(objName) + ".sql"
				filePath := filepath.Join(dirPath, fileName)
				relativePath := filepath.Join(dir, fileName)

				// Write object file
				if err := writeObjectFile(filePath, objSteps, targetSchema); err != nil {
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
	header := generateDumpHeader(schemaIR)
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

// writeObjectFile writes a single object file with its SQL statements
func writeObjectFile(filePath string, steps []diff.PlanStep, targetSchema string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for i, step := range steps {
		isComment := strings.ToUpper(step.ObjectType) == "COMMENT"

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
				prevIsComment := strings.ToUpper(steps[i-1].ObjectType) == "COMMENT"
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
			commentSchemaName := step.ObjectPath
			if strings.Contains(step.ObjectPath, ".") {
				parts := strings.Split(step.ObjectPath, ".")
				if len(parts) >= 2 && parts[0] == targetSchema {
					commentSchemaName = "-"
				} else {
					commentSchemaName = parts[0]
				}
			}

			// Print object comment header
			objectName := step.ObjectPath
			if strings.Contains(step.ObjectPath, ".") {
				parts := strings.Split(step.ObjectPath, ".")
				if len(parts) >= 2 {
					objectName = parts[1]
				}
			}

			file.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: -\n", objectName, strings.ToUpper(step.ObjectType), commentSchemaName))
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
func getObjectDirectory(objectType string) string {
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
		// Comments handled separately in createMultiFileOutput
		return "tables" // fallback, will be overridden
	default:
		return "misc"
	}
}

// getObjectName extracts the object name from the object path
func getObjectName(objectPath string) string {
	if strings.Contains(objectPath, ".") {
		parts := strings.Split(objectPath, ".")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return objectPath
}

// getGroupingName determines the appropriate name for grouping objects into files
func getGroupingName(step diff.PlanStep, targetSchema string) string {
	objType := strings.ToUpper(step.ObjectType)

	// For table-related objects, try to extract the table name from SourceChange
	switch objType {
	case "INDEX", "CONSTRAINT", "POLICY", "TRIGGER", "RULE":
		if tableName := extractTableNameFromContext(step); tableName != "" {
			return tableName
		}
	case "COMMENT":
		// For comments, we need to determine the parent object
		// For index comments, group with parent table
		if step.SourceChange != nil {
			switch obj := step.SourceChange.(type) {
			case *ir.Index:
				return obj.Table // Group index comments with their table
			case *ir.Table:
				return obj.Name // Table and column comments group with table
			case *ir.View:
				return obj.Name // View comments group with view
			}
		}
		// Fallback: extract parent object name from object path
		// ObjectPath format: "schema.table" or "schema.table.column" for table/column comments
		// ObjectPath format: "schema.view" for view comments
		if parts := strings.Split(step.ObjectPath, "."); len(parts) >= 2 {
			return parts[1] // Return parent object name (table/view)
		}
	}

	// For standalone objects or if table name extraction fails, use object name
	return getObjectName(step.ObjectPath)
}

// extractTableNameFromContext extracts table name from the SourceChange in SQLContext
func extractTableNameFromContext(step diff.PlanStep) string {
	if step.SourceChange == nil {
		return ""
	}

	// Try to extract table name based on the type of SourceChange
	switch obj := step.SourceChange.(type) {
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
func getCommentParentDirectory(step diff.PlanStep) string {
	if step.SourceChange != nil {
		switch step.SourceChange.(type) {
		case *ir.View:
			return "views"
		case *ir.Table, *ir.Index:
			return "tables"
		}
	}
	// Fallback to tables for unknown comment types
	return "tables"
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

func runDump(cmd *cobra.Command, args []string) error {
	// Validate flags
	if multiFile && file == "" {
		// When --multi-file is used but no --file specified, emit warning and use single-file mode
		fmt.Fprintf(os.Stderr, "Warning: --multi-file flag requires --file to be specified. Fallback to single-file mode.\n")
		multiFile = false
	}

	// Derive final password: use flag if provided, otherwise check environment variable
	finalPassword := password
	if finalPassword == "" {
		if envPassword := os.Getenv("PGPASSWORD"); envPassword != "" {
			finalPassword = envPassword
		}
	}

	// Build database connection
	config := &util.ConnectionConfig{
		Host:            host,
		Port:            port,
		Database:        db,
		User:            user,
		Password:        finalPassword,
		SSLMode:         "prefer",
		ApplicationName: "pgschema",
	}

	dbConn, err := util.Connect(config)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	ctx := context.Background()

	// Build IR using the IR system
	inspector := ir.NewInspector(dbConn)
	schemaIR, err := inspector.BuildIR(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to build IR: %w", err)
	}

	// Create SQLCollector to collect all SQL statements
	collector := diff.NewSQLCollector()

	// Generate dump SQL using collector
	diff.CollectDumpSQL(schemaIR, schema, collector)

	if multiFile {
		// Multi-file mode - output to files
		err := createMultiFileOutput(collector, schemaIR, schema, file)
		if err != nil {
			return fmt.Errorf("failed to create multi-file output: %w", err)
		}
	} else {
		// Single file mode - output to stdout

		// Generate and print header
		header := generateDumpHeader(schemaIR)
		fmt.Print(header)

		// Print all SQL statements from collector with proper separators
		steps := collector.GetSteps()
		for i, step := range steps {
			// Check if this is a comment statement
			if strings.ToUpper(step.ObjectType) == "COMMENT" {
				// For comments, just write the raw SQL without DDL header
				if i > 0 {
					fmt.Print("\n") // Add separator from previous statement
				}
				fmt.Print(step.SQL)
				if !strings.HasSuffix(step.SQL, "\n") {
					fmt.Print("\n")
				}
			} else {
				// Add DDL separator with comment header for non-comment statements
				fmt.Print("--\n")

				// Determine schema name for comment (use "-" for target schema)
				commentSchemaName := step.ObjectPath
				if strings.Contains(step.ObjectPath, ".") {
					parts := strings.Split(step.ObjectPath, ".")
					if len(parts) >= 2 && parts[0] == schema {
						commentSchemaName = "-"
					} else {
						commentSchemaName = parts[0]
					}
				}

				// Print object comment header
				objectName := step.ObjectPath
				if strings.Contains(step.ObjectPath, ".") {
					parts := strings.Split(step.ObjectPath, ".")
					if len(parts) >= 2 {
						objectName = parts[1]
					}
				}

				fmt.Printf("-- Name: %s; Type: %s; Schema: %s; Owner: -\n", objectName, strings.ToUpper(step.ObjectType), commentSchemaName)
				fmt.Print("--\n")
				fmt.Print("\n")

				// Print the SQL statement
				fmt.Print(step.SQL)
			}

			// Add newline after SQL, and extra newline only if not last item
			if i < len(steps)-1 {
				fmt.Print("\n\n")
			}
		}
	}

	return nil
}
