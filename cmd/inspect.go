package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pgschema/pgschema/internal/queries"
	"github.com/spf13/cobra"
)

var dsn string

var InspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect database schema",
	Long:  "Inspect and output database schema information including schemas and tables",
	RunE:  runInspect,
}

func init() {
	InspectCmd.Flags().StringVar(&dsn, "dsn", "", "Database connection string (required)")
	InspectCmd.MarkFlagRequired("dsn")
}

// DatabaseObject represents a database object for dependency sorting
type DatabaseObject struct {
	Schema      string
	Name        string
	Type        string // "table", "view"
	FullName    string // schema.name
	TableRow    *queries.GetTablesRow
	Dependencies []string // List of objects this depends on (schema.name format)
}

// topologicalSort performs topological sorting on database objects
func topologicalSort(objects []DatabaseObject) []DatabaseObject {
	// Build adjacency list and in-degree count
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)
	objectMap := make(map[string]DatabaseObject)
	
	// Initialize all objects
	for _, obj := range objects {
		inDegree[obj.FullName] = 0
		adjList[obj.FullName] = []string{}
		objectMap[obj.FullName] = obj
	}
	
	// Build dependency graph
	for _, obj := range objects {
		for _, dep := range obj.Dependencies {
			if _, exists := objectMap[dep]; exists {
				adjList[dep] = append(adjList[dep], obj.FullName)
				inDegree[obj.FullName]++
			}
		}
	}
	
	// Kahn's algorithm for topological sorting
	var queue []string
	var result []DatabaseObject
	
	// Start with objects that have no dependencies
	for objName, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, objName)
		}
	}
	
	// Sort queue for deterministic output
	sort.Strings(queue)
	
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, objectMap[current])
		
		// Reduce in-degree of dependent objects
		for _, neighbor := range adjList[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
				sort.Strings(queue) // Keep queue sorted for deterministic output
			}
		}
	}
	
	// Check for cycles (should not happen with proper database design)
	if len(result) != len(objects) {
		logger.Debug("Detected dependency cycle, falling back to original order")
		return objects
	}
	
	return result
}

func runInspect(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	logger.Debug("Starting inspect command", "dsn", dsn)
	
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()
	logger.Debug("Database connection opened successfully")

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	logger.Debug("Database ping successful")

	q := queries.New(db)

	// Get database version
	logger.Debug("Getting database version...")
	var dbVersion string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&dbVersion); err != nil {
		return fmt.Errorf("failed to get database version: %w", err)
	}
	logger.Debug("Database version retrieved", "version", dbVersion)
	
	// Extract version number from the version string
	if strings.Contains(dbVersion, "PostgreSQL") {
		parts := strings.Fields(dbVersion)
		if len(parts) >= 2 {
			dbVersion = "PostgreSQL " + parts[1]
		}
	}
	logger.Debug("Formatted database version", "formatted_version", dbVersion)

	// Get pgschema version
	version := "0.0.1" // default
	if versionBytes, err := os.ReadFile("VERSION"); err == nil {
		version = strings.TrimSpace(string(versionBytes))
	}
	logger.Debug("pgschema version", "version", version)
	
	// Print header for schema inspection (no SET commands needed for inspection)
	fmt.Println("--")
	fmt.Println("-- PostgreSQL database dump")
	fmt.Println("--")
	fmt.Println("")
	fmt.Printf("-- Dumped from database version %s\n", dbVersion)
	fmt.Printf("-- Dumped by pgschema version %s\n", version)
	fmt.Println("")

	// Get and process all data
	logger.Debug("Starting to query database objects...")
	
	logger.Debug("Querying schemas...")
	schemas, err := q.GetSchemas(ctx)
	if err != nil {
		return fmt.Errorf("failed to get schemas: %w", err)
	}
	logger.Debug("Found schemas", "count", len(schemas), "schemas", schemas)

	logger.Debug("Querying sequences...")
	sequences, err := q.GetSequences(ctx)
	if err != nil {
		return fmt.Errorf("failed to get sequences: %w", err)
	}
	logger.Debug("Found sequences", "count", len(sequences))

	logger.Debug("Querying functions...")
	functions, err := q.GetFunctions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get functions: %w", err)
	}
	logger.Debug("Found functions", "count", len(functions))

	logger.Debug("Querying tables...")
	tables, err := q.GetTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}
	logger.Debug("Found tables", "count", len(tables))

	logger.Debug("Querying columns...")
	columns, err := q.GetColumns(ctx)
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}
	logger.Debug("Found columns", "count", len(columns))

	logger.Debug("Querying constraints...")
	constraints, err := q.GetConstraints(ctx)
	if err != nil {
		return fmt.Errorf("failed to get constraints: %w", err)
	}
	logger.Debug("Found constraints", "count", len(constraints))

	logger.Debug("Querying views...")
	views, err := q.GetViews(ctx)
	if err != nil {
		return fmt.Errorf("failed to get views: %w", err)
	}
	logger.Debug("Found views", "count", len(views))

	logger.Debug("Querying triggers...")
	triggers, err := q.GetTriggers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get triggers: %w", err)
	}
	logger.Debug("Found triggers", "count", len(triggers))
	
	logger.Debug("Querying view dependencies...")
	viewDeps, err := q.GetViewDependencies(ctx)
	if err != nil {
		return fmt.Errorf("failed to get view dependencies: %w", err)
	}
	logger.Debug("Found view dependencies", "count", len(viewDeps))

	// Build dependency-sorted table and view list (only for structural dependencies, not foreign keys)
	logger.Debug("Building dependency graph for structural dependencies...")
	var dbObjects []DatabaseObject
	
	// Build view dependency map (only structural dependencies matter for creation order)
	viewDepMap := make(map[string][]string) // view -> list of tables/views it depends on
	for _, dep := range viewDeps {
		depSchema := fmt.Sprintf("%s", dep.DependentSchema)
		depName := fmt.Sprintf("%s", dep.DependentName)
		srcSchema := fmt.Sprintf("%s", dep.SourceSchema)
		srcName := fmt.Sprintf("%s", dep.SourceName)
		
		viewKey := fmt.Sprintf("%s.%s", depSchema, depName)
		sourceKey := fmt.Sprintf("%s.%s", srcSchema, srcName)
		viewDepMap[viewKey] = append(viewDepMap[viewKey], sourceKey)
	}
	
	// Create DatabaseObject list
	for _, table := range tables {
		schemaName := fmt.Sprintf("%s", table.TableSchema)
		tableName := fmt.Sprintf("%s", table.TableName)
		tableType := fmt.Sprintf("%s", table.TableType)
		fullName := fmt.Sprintf("%s.%s", schemaName, tableName)
		
		var deps []string
		if tableType == "VIEW" {
			// Only views have structural dependencies for creation order
			// Foreign key dependencies are handled separately via ALTER TABLE later
			deps = append(deps, viewDepMap[fullName]...)
		}
		// BASE TABLEs have no structural dependencies for creation (foreign keys added later)
		
		dbObjects = append(dbObjects, DatabaseObject{
			Schema:       schemaName,
			Name:         tableName,
			Type:         strings.ToLower(tableType),
			FullName:     fullName,
			TableRow:     &table,
			Dependencies: deps,
		})
	}
	
	// Sort objects by structural dependencies only (views must come after their tables)
	sortedObjects := topologicalSort(dbObjects)
	logger.Debug("Sorted objects by structural dependencies", "count", len(sortedObjects))

	// Step 1: Create schemas (skip public schema)
	logger.Debug("Processing schemas...")
	for _, schema := range schemas {
		schemaName := fmt.Sprintf("%s", schema)
		if schemaName != "public" {
			logger.Debug("Creating schema", "schema", schemaName)
			printComment("SCHEMA", schemaName, schemaName, "")
			fmt.Printf("CREATE SCHEMA %s;\n", schemaName)
			fmt.Println("")
		} else {
			logger.Debug("Skipping public schema")
		}
	}

	// Step 2: Create functions
	for _, fn := range functions {
		schemaName := fmt.Sprintf("%s", fn.RoutineSchema)
		functionName := fmt.Sprintf("%s", fn.RoutineName)
		functionDef := fmt.Sprintf("%s", fn.RoutineDefinition)
		language := fmt.Sprintf("%s", fn.ExternalLanguage)
		
		if functionDef != "<nil>" && functionDef != "" {
			printComment("FUNCTION", fmt.Sprintf("%s()", functionName), schemaName, "")
			fmt.Printf("CREATE FUNCTION %s.%s() RETURNS trigger\n", schemaName, functionName)
			fmt.Printf("    LANGUAGE %s\n", strings.ToLower(language))
			fmt.Printf("    AS $$%s$$;\n", functionDef)
			fmt.Println("")
		}
	}

	// Group columns by table
	tableColumns := make(map[string][]queries.GetColumnsRow)
	for _, col := range columns {
		tableKey := fmt.Sprintf("%s.%s", col.TableSchema, col.TableName)
		tableColumns[tableKey] = append(tableColumns[tableKey], col)
	}

	// Step 3: Create tables and sequences (following pg_dump pattern with dependency order)
	logger.Debug("Processing tables and sequences in dependency order...")
	for _, obj := range sortedObjects {
		schemaName := obj.Schema
		tableName := obj.Name
		tableType := strings.ToUpper(obj.Type)
		
		logger.Debug("Processing object", "schema", schemaName, "name", tableName, "type", tableType, "deps", obj.Dependencies)
		
		// Handle both base tables and views in this section (like pg_dump)
		tableKey := fmt.Sprintf("%s.%s", schemaName, tableName)
		tableCols := tableColumns[tableKey]
		
		if tableType == "BASE TABLE" {
			// Create table (minimal, no defaults)
			printComment("TABLE", tableName, schemaName, "")
			fmt.Printf("CREATE TABLE %s.%s (\n", schemaName, tableName)
			
			// Add columns (without defaults)
			for i, col := range tableCols {
				if i > 0 {
					fmt.Printf(",\n")
				}
				
				colName := fmt.Sprintf("%s", col.ColumnName)
				dataType := fmt.Sprintf("%s", col.DataType)
				isNullable := fmt.Sprintf("%s", col.IsNullable)
				
				// Build column definition (no defaults, no sequences)
				fmt.Printf("    %s %s", colName, dataType)
				
				// Add length/precision for specific types
				if col.CharacterMaximumLength != nil {
					if maxLen, ok := col.CharacterMaximumLength.(int64); ok {
						fmt.Printf("(%d)", maxLen)
					}
				} else if col.NumericPrecision != nil && col.NumericScale != nil && dataType != "integer" && dataType != "bigint" && dataType != "smallint" {
					// Only add precision/scale for decimal/numeric types, not for integer types
					if precision, okP := col.NumericPrecision.(int64); okP {
						if scale, okS := col.NumericScale.(int64); okS {
							fmt.Printf("(%d,%d)", precision, scale)
						}
					}
				}
				
				// Add NOT NULL constraint
				if isNullable == "NO" {
					fmt.Printf(" NOT NULL")
				}
				
				// Add non-sequence defaults inline (like pg_dump does)
				if col.ColumnDefault != nil {
					defaultVal := fmt.Sprintf("%s", col.ColumnDefault)
					if defaultVal != "<nil>" && defaultVal != "" && !strings.Contains(defaultVal, "nextval") {
						fmt.Printf(" DEFAULT %s", defaultVal)
					}
				}
			}
			
			// Add meaningful CHECK constraints only (no NOT NULL checks)
			for _, constraint := range constraints {
				constSchema := fmt.Sprintf("%s", constraint.TableSchema)
				constTable := fmt.Sprintf("%s", constraint.TableName)
				if constSchema == schemaName && constTable == tableName {
					constraintType := fmt.Sprintf("%s", constraint.ConstraintType)
					if constraintType == "CHECK" {
						constraintName := fmt.Sprintf("%s", constraint.ConstraintName)
						checkClause := fmt.Sprintf("%s", constraint.CheckClause)
						if checkClause != "<nil>" && checkClause != "" {
							// Skip NOT NULL check constraints since we handle NOT NULL inline
							if !strings.Contains(strings.ToLower(checkClause), "is not null") && !strings.Contains(constraintName, "_not_null") {
								fmt.Printf(",\n    CONSTRAINT %s CHECK (%s)", constraintName, checkClause)
							}
						}
					}
				}
			}
			
			fmt.Printf("\n);\n")
			fmt.Println("")
			
			// Immediately create related sequences after each table (pg_dump pattern)
			// Find sequences that belong to this table by checking column defaults
			for _, col := range tableCols {
				if col.ColumnDefault != nil {
					defaultVal := fmt.Sprintf("%s", col.ColumnDefault)
					if strings.Contains(defaultVal, "nextval") {
						// Extract sequence name from default value like "nextval('audit_id_seq'::regclass)"
						for _, seq := range sequences {
							seqSchema := fmt.Sprintf("%s", seq.SequenceSchema)
							seqName := fmt.Sprintf("%s", seq.SequenceName)
							
							if strings.Contains(defaultVal, seqName) && seqSchema == schemaName {
								dataType := fmt.Sprintf("%s", seq.DataType)
								startValue := fmt.Sprintf("%s", seq.StartValue)
								minValue := fmt.Sprintf("%s", seq.MinimumValue)
								maxValue := fmt.Sprintf("%s", seq.MaximumValue)
								increment := fmt.Sprintf("%s", seq.Increment)
								
								printComment("SEQUENCE", seqName, seqSchema, "")
								fmt.Printf("CREATE SEQUENCE %s.%s\n", seqSchema, seqName)
								if dataType != "<nil>" && dataType != "bigint" {
									fmt.Printf("    AS %s\n", dataType)
								}
								fmt.Printf("    START WITH %s\n", startValue)
								fmt.Printf("    INCREMENT BY %s\n", increment)
								
								// Handle min/max values
								if minValue == "1" {
									fmt.Printf("    NO MINVALUE\n")
								} else {
									fmt.Printf("    MINVALUE %s\n", minValue)
								}
								
								// Check for both bigint and integer max values to output NO MAXVALUE
								if maxValue == strconv.FormatInt(math.MaxInt64, 10) || maxValue == strconv.FormatInt(math.MaxInt32, 10) {
									fmt.Printf("    NO MAXVALUE\n")
								} else {
									fmt.Printf("    MAXVALUE %s\n", maxValue)
								}
								
								fmt.Printf("    CACHE 1;\n")
								fmt.Println("")
								
								// Add OWNED BY immediately after sequence creation (pg_dump pattern)
								colName := fmt.Sprintf("%s", col.ColumnName)
								printComment("SEQUENCE OWNED BY", seqName, seqSchema, "")
								fmt.Printf("ALTER SEQUENCE %s.%s OWNED BY %s.%s.%s;\n", seqSchema, seqName, schemaName, tableName, colName)
								fmt.Println("")
								break // Found the sequence for this column
							}
						}
					}
				}
			}
			
		} else if tableType == "VIEW" {
			// Handle views
			for _, view := range views {
				viewSchema := fmt.Sprintf("%s", view.TableSchema)
				viewName := fmt.Sprintf("%s", view.TableName)
				viewDef := fmt.Sprintf("%s", view.ViewDefinition)
				
				if viewSchema == schemaName && viewName == tableName && viewDef != "<nil>" && viewDef != "" {
					printComment("VIEW", viewName, viewSchema, "")
					// Remove any trailing semicolons from viewDef, we'll add our own
					cleanViewDef := strings.TrimSpace(viewDef)
					for strings.HasSuffix(cleanViewDef, ";") {
						cleanViewDef = strings.TrimSuffix(cleanViewDef, ";")
						cleanViewDef = strings.TrimSpace(cleanViewDef)
					}
					// Add schema qualifiers to table references in the view definition
					processedViewDef := addSchemaQualifiersToView(cleanViewDef, viewSchema)
					fmt.Printf("CREATE VIEW %s.%s AS\n%s;\n", viewSchema, viewName, processedViewDef)
					fmt.Println("")
				}
			}
		}
	}

	// Step 4: Add column defaults for sequences (ALTER TABLE ... ALTER COLUMN ... SET DEFAULT)
	processedDefaults := make(map[string]bool)
	for _, col := range columns {
		if col.ColumnDefault != nil {
			defaultVal := fmt.Sprintf("%s", col.ColumnDefault)
			if strings.Contains(defaultVal, "nextval") {
				schemaName := fmt.Sprintf("%s", col.TableSchema)
				tableName := fmt.Sprintf("%s", col.TableName)
				columnName := fmt.Sprintf("%s", col.ColumnName)
				key := fmt.Sprintf("%s.%s.%s", schemaName, tableName, columnName)
				
				if !processedDefaults[key] {
					printComment("DEFAULT", fmt.Sprintf("%s %s", tableName, columnName), schemaName, "")
					// Add schema qualifiers to the default value (for nextval sequences)
					qualifiedDefaultVal := addSchemaQualifiers(defaultVal, schemaName)
					fmt.Printf("ALTER TABLE ONLY %s.%s ALTER COLUMN %s SET DEFAULT %s;\n", 
						schemaName, tableName, columnName, qualifiedDefaultVal)
					fmt.Println("")
					fmt.Println("")
					processedDefaults[key] = true
				}
			}
		}
	}

	// Step 5: Add PRIMARY KEY and UNIQUE constraints
	for _, constraint := range constraints {
		schemaName := fmt.Sprintf("%s", constraint.TableSchema)
		tableName := fmt.Sprintf("%s", constraint.TableName)
		constraintType := fmt.Sprintf("%s", constraint.ConstraintType)
		constraintName := fmt.Sprintf("%s", constraint.ConstraintName)
		
		switch constraintType {
		case "PRIMARY KEY":
			columnName := fmt.Sprintf("%s", constraint.ColumnName)
			printComment("CONSTRAINT", fmt.Sprintf("%s %s", tableName, constraintName), schemaName, "")
			fmt.Printf("ALTER TABLE ONLY %s.%s\n", schemaName, tableName)
			fmt.Printf("    ADD CONSTRAINT %s PRIMARY KEY (%s);\n", constraintName, columnName)
			fmt.Println("")
			fmt.Println("")
		case "UNIQUE":
			columnName := fmt.Sprintf("%s", constraint.ColumnName)
			printComment("CONSTRAINT", fmt.Sprintf("%s %s", tableName, constraintName), schemaName, "")
			fmt.Printf("ALTER TABLE ONLY %s.%s\n", schemaName, tableName)
			fmt.Printf("    ADD CONSTRAINT %s UNIQUE (%s);\n", constraintName, columnName)
			fmt.Println("")
			fmt.Println("")
		}
	}

	// Step 6: Add triggers (group by trigger name, table, timing, and statement to combine events)
	type triggerKey struct {
		schema    string
		name      string
		table     string
		timing    string
		statement string
	}
	triggerGroups := make(map[triggerKey][]string)
	
	// Group triggers by key and collect events
	for _, trigger := range triggers {
		schemaName := fmt.Sprintf("%s", trigger.TriggerSchema)
		triggerName := fmt.Sprintf("%s", trigger.TriggerName)
		tableName := fmt.Sprintf("%s", trigger.EventObjectTable)
		timing := fmt.Sprintf("%s", trigger.ActionTiming)
		event := fmt.Sprintf("%s", trigger.EventManipulation)
		statement := fmt.Sprintf("%s", trigger.ActionStatement)
		
		if statement != "<nil>" && statement != "" {
			key := triggerKey{
				schema:    schemaName,
				name:      triggerName,
				table:     tableName,
				timing:    timing,
				statement: statement,
			}
			triggerGroups[key] = append(triggerGroups[key], event)
		}
	}
	
	// Output combined triggers
	for key, events := range triggerGroups {
		printComment("TRIGGER", key.name, key.schema, "")
		eventList := strings.Join(events, " OR ")
		fmt.Printf("CREATE TRIGGER %s %s %s ON %s.%s FOR EACH ROW %s;\n",
			key.name, key.timing, eventList, key.schema, key.table, key.statement)
		fmt.Println("")
		fmt.Println("")
	}

	// Step 7: Add FOREIGN KEY constraints
	for _, constraint := range constraints {
		schemaName := fmt.Sprintf("%s", constraint.TableSchema)
		tableName := fmt.Sprintf("%s", constraint.TableName)
		constraintType := fmt.Sprintf("%s", constraint.ConstraintType)
		constraintName := fmt.Sprintf("%s", constraint.ConstraintName)
		
		if constraintType == "FOREIGN KEY" {
			columnName := fmt.Sprintf("%s", constraint.ColumnName)
			foreignTable := fmt.Sprintf("%s", constraint.ForeignTableName)
			foreignColumn := fmt.Sprintf("%s", constraint.ForeignColumnName)
			foreignSchema := fmt.Sprintf("%s", constraint.ForeignTableSchema)
			if foreignTable != "<nil>" && foreignColumn != "<nil>" {
				// Build referential actions
				var referentialActions []string
				
				deleteRule := fmt.Sprintf("%s", constraint.DeleteRule)
				if deleteRule != "<nil>" && deleteRule != "NO ACTION" && deleteRule != "" {
					referentialActions = append(referentialActions, fmt.Sprintf("ON DELETE %s", deleteRule))
				}
				
				updateRule := fmt.Sprintf("%s", constraint.UpdateRule)
				if updateRule != "<nil>" && updateRule != "NO ACTION" && updateRule != "" {
					referentialActions = append(referentialActions, fmt.Sprintf("ON UPDATE %s", updateRule))
				}
				
				printComment("FK CONSTRAINT", fmt.Sprintf("%s %s", tableName, constraintName), schemaName, "")
				fmt.Printf("ALTER TABLE ONLY %s.%s\n", schemaName, tableName)
				if len(referentialActions) > 0 {
					fmt.Printf("    ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s.%s(%s) %s;\n", 
						constraintName, columnName, foreignSchema, foreignTable, foreignColumn, strings.Join(referentialActions, " "))
				} else {
					fmt.Printf("    ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s.%s(%s);\n", 
						constraintName, columnName, foreignSchema, foreignTable, foreignColumn)
				}
				fmt.Println("")
				fmt.Println("")
			}
		}
	}

	// Final comment
	fmt.Println("--")
	fmt.Println("-- PostgreSQL database dump complete")
	fmt.Println("--")
	fmt.Println("")

	return nil
}

// addSchemaQualifiers adds schema qualifiers to object references in SQL text
func addSchemaQualifiers(sqlText, schemaName string) string {
	result := sqlText
	
	// Handle view definitions - table references after FROM and JOIN keywords
	viewKeywords := []string{"FROM ", "JOIN ", "from ", "join "}
	for _, keyword := range viewKeywords {
		result = addSchemaQualifiersForKeyword(result, keyword, schemaName)
	}
	
	// Handle sequence references in nextval() calls
	result = addSchemaQualifiersToNextval(result, schemaName)
	
	return result
}

// addSchemaQualifiersForKeyword processes table references after specific SQL keywords
func addSchemaQualifiersForKeyword(sqlText, keyword, schemaName string) string {
	parts := strings.Split(sqlText, keyword)
	if len(parts) <= 1 {
		return sqlText
	}
	
	// Process each part after the keyword
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		
		// Handle cases with parentheses like "FROM (dept_emp d"
		if strings.HasPrefix(strings.TrimSpace(part), "(") {
			// Find the table name after the opening parenthesis
			trimmed := strings.TrimSpace(part)
			afterParen := trimmed[1:] // Remove the opening parenthesis
			words := strings.Fields(afterParen)
			if len(words) > 0 {
				tableName := words[0]
				// Only add schema qualifier if the table name doesn't already have one
				if !strings.Contains(tableName, ".") {
					// Replace the unqualified table name with schema-qualified name
					words[0] = fmt.Sprintf("%s.%s", schemaName, tableName)
					// Reconstruct the part with the parenthesis
					parts[i] = fmt.Sprintf("(%s", strings.Join(words, " "))
				}
			}
		} else {
			// Regular case without parentheses
			words := strings.Fields(part)
			if len(words) > 0 {
				tableName := words[0]
				// Only add schema qualifier if the table name doesn't already have one
				if !strings.Contains(tableName, ".") {
					// Replace the unqualified table name with schema-qualified name
					words[0] = fmt.Sprintf("%s.%s", schemaName, tableName)
					parts[i] = strings.Join(words, " ")
				}
			}
		}
	}
	
	return strings.Join(parts, keyword)
}

// addSchemaQualifiersToNextval adds schema qualifiers to sequence names in nextval() calls
func addSchemaQualifiersToNextval(sqlText, schemaName string) string {
	// Pattern: nextval('sequence_name'::regclass) -> nextval('schema.sequence_name'::regclass)
	// Use a simple string replacement approach for safety
	
	// Find all nextval() calls
	result := sqlText
	nextvalStart := "nextval('"
	
	for {
		startIdx := strings.Index(result, nextvalStart)
		if startIdx == -1 {
			break
		}
		
		// Find the end of the sequence name (look for the closing quote)
		nameStart := startIdx + len(nextvalStart)
		endIdx := strings.Index(result[nameStart:], "'")
		if endIdx == -1 {
			break
		}
		endIdx += nameStart
		
		// Extract the sequence name
		seqName := result[nameStart:endIdx]
		
		// Only add schema qualifier if it doesn't already have one
		if !strings.Contains(seqName, ".") {
			qualifiedName := fmt.Sprintf("%s.%s", schemaName, seqName)
			result = result[:nameStart] + qualifiedName + result[endIdx:]
		}
		
		// Move past this nextval call to find the next one
		result = result[:startIdx] + strings.Replace(result[startIdx:], nextvalStart, "NEXTVAL_PROCESSED('", 1)
	}
	
	// Restore the original nextval keyword
	result = strings.ReplaceAll(result, "NEXTVAL_PROCESSED(", "nextval(")
	
	return result
}

// addSchemaQualifiersToView adds schema qualifiers to table references in view definitions (for backward compatibility)
func addSchemaQualifiersToView(viewDef, schemaName string) string {
	return addSchemaQualifiers(viewDef, schemaName)
}

// printComment prints a pg_dump style comment for database objects with proper spacing
func printComment(objectType, objectName, schemaName, owner string) {
	// Always ensure there's a blank line before the comment (except for the very first object)
	fmt.Println("")
	fmt.Println("--")
	if owner != "" {
		fmt.Printf("-- Name: %s; Type: %s; Schema: %s; Owner: %s\n", objectName, objectType, schemaName, owner)
	} else {
		fmt.Printf("-- Name: %s; Type: %s; Schema: %s; Owner: -\n", objectName, objectType, schemaName)
	}
	fmt.Println("--")
	fmt.Println("")
}