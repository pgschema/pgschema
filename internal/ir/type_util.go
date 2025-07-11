package ir

import "strings"

// NormalizePostgreSQLType converts PostgreSQL internal type names to their canonical SQL standard names.
// This function handles:
// - Internal type names (int4 -> integer, bool -> boolean)
// - pg_catalog prefixed types (pg_catalog.int4 -> integer)
// - Array types (_text -> text[], _int4 -> integer[])
// - Verbose type names (timestamp with time zone -> timestamptz)
// - Serial types to uppercase (serial -> SERIAL)
func NormalizePostgreSQLType(typeName string) string {
	// Main type mapping table
	typeMap := map[string]string{
		// Numeric types
		"int2":               "smallint",
		"int4":               "integer",
		"int8":               "bigint",
		"float4":             "real",
		"float8":             "double precision",
		"bool":               "boolean",
		"pg_catalog.int2":    "smallint",
		"pg_catalog.int4":    "integer",
		"pg_catalog.int8":    "bigint",
		"pg_catalog.float4":  "real",
		"pg_catalog.float8":  "double precision",
		"pg_catalog.bool":    "boolean",
		"pg_catalog.numeric": "numeric",

		// Character types
		"bpchar":             "character",
		"varchar":            "character varying",
		"pg_catalog.text":    "text",
		"pg_catalog.varchar": "character varying",
		"pg_catalog.bpchar":  "character",

		// Date/time types - convert verbose forms to canonical short forms
		"timestamp with time zone":    "timestamptz",
		"time with time zone":         "timetz",
		"timestamptz":                 "timestamptz",
		"timetz":                      "timetz",
		"pg_catalog.timestamptz":      "timestamptz",
		"pg_catalog.timestamp":        "timestamp",
		"pg_catalog.date":             "date",
		"pg_catalog.time":             "time",
		"pg_catalog.timetz":           "timetz",
		"pg_catalog.interval":         "interval",

		// Array types (internal PostgreSQL array notation)
		"_text":     "text[]",
		"_int2":     "smallint[]",
		"_int4":     "integer[]",
		"_int8":     "bigint[]",
		"_float4":   "real[]",
		"_float8":   "double precision[]",
		"_bool":     "boolean[]",
		"_varchar":  "character varying[]",
		"_char":     "character[]",
		"_bpchar":   "character[]",
		"_numeric":  "numeric[]",
		"_uuid":     "uuid[]",
		"_json":     "json[]",
		"_jsonb":    "jsonb[]",
		"_bytea":    "bytea[]",
		"_inet":     "inet[]",
		"_cidr":     "cidr[]",
		"_macaddr":  "macaddr[]",
		"_macaddr8": "macaddr8[]",
		"_date":     "date[]",
		"_time":     "time[]",
		"_timetz":   "timetz[]",
		"_timestamp": "timestamp[]",
		"_timestamptz": "timestamptz[]",
		"_interval": "interval[]",

		// Other common types
		"pg_catalog.uuid":    "uuid",
		"pg_catalog.json":    "json",
		"pg_catalog.jsonb":   "jsonb",
		"pg_catalog.bytea":   "bytea",
		"pg_catalog.inet":    "inet",
		"pg_catalog.cidr":    "cidr",
		"pg_catalog.macaddr": "macaddr",

		// Serial types (keep as uppercase for SQL generation)
		"serial":      "SERIAL",
		"smallserial": "SMALLSERIAL",
		"bigserial":   "BIGSERIAL",
	}

	// Check if we have a direct mapping
	if normalized, exists := typeMap[typeName]; exists {
		return normalized
	}

	// Remove pg_catalog prefix for unmapped types
	if strings.HasPrefix(typeName, "pg_catalog.") {
		return strings.TrimPrefix(typeName, "pg_catalog.")
	}

	// Return as-is if no mapping found
	return typeName
}

// StripSchemaPrefix removes the schema prefix from a type name if it matches the target schema
func StripSchemaPrefix(typeName, targetSchema string) string {
	if typeName == "" || targetSchema == "" {
		return typeName
	}

	// Check if the type has the target schema prefix
	prefix := targetSchema + "."
	if strings.HasPrefix(typeName, prefix) {
		return strings.TrimPrefix(typeName, prefix)
	}

	return typeName
}