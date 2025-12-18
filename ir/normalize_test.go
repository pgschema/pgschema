package ir

import (
	"testing"
)

func TestIsBuiltInType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		expected bool
	}{
		// Basic built-in types
		{"integer", "integer", true},
		{"text", "text", true},
		{"varchar", "varchar", true},
		{"boolean", "boolean", true},
		{"timestamp", "timestamp", true},
		{"uuid", "uuid", true},
		{"jsonb", "jsonb", true},

		// Types with parentheses
		{"varchar with length", "varchar(255)", true},
		{"character varying with length", "character varying(100)", true},
		{"char with length", "char(10)", true},
		{"numeric with precision", "numeric(10,2)", true},
		{"decimal with precision", "decimal(5,2)", true},
		{"bit with length", "bit(8)", true},
		{"bit varying with length", "bit varying(64)", true},
		{"timestamp with precision", "timestamp(6)", true},
		{"time with precision", "time(3)", true},
		{"interval with fields", "interval(2)", true},

		// Array types
		{"text array", "text[]", true},
		{"varchar array", "varchar[]", true},
		{"integer array", "integer[]", true},
		{"uuid array", "uuid[]", true},
		{"jsonb array", "jsonb[]", true},
		{"varchar with length array", "varchar(255)[]", true},
		{"numeric with precision array", "numeric(10,2)[]", true},

		// Types with pg_catalog prefix
		{"pg_catalog.text", "pg_catalog.text", true},
		{"pg_catalog.varchar", "pg_catalog.varchar", true},
		{"pg_catalog.int4", "pg_catalog.int4", true},
		{"pg_catalog.bool", "pg_catalog.bool", true},
		{"pg_catalog.varchar with length", "pg_catalog.varchar(100)", true},
		{"pg_catalog.text array", "pg_catalog.text[]", true},

		// Case sensitivity (should be case-insensitive)
		{"TEXT uppercase", "TEXT", true},
		{"VARCHAR uppercase", "VARCHAR", true},
		{"Integer mixed case", "Integer", true},
		{"BOOLEAN uppercase", "BOOLEAN", true},
		{"VarChar mixed case", "VarChar(50)", true},

		// Internal type names
		{"int2", "int2", true},
		{"int4", "int4", true},
		{"int8", "int8", true},
		{"float4", "float4", true},
		{"float8", "float8", true},
		{"bool", "bool", true},
		{"bpchar", "bpchar", true},

		// Custom types (should return false)
		{"custom enum", "status_enum", false},
		{"custom type", "my_custom_type", false},
		{"user defined", "address_type", false},
		{"custom with underscore", "user_status", false},

		// Edge cases
		{"empty string", "", false},
		{"whitespace", "   ", false},
		{"unknown type", "notarealtype", false},
		{"type with schema prefix", "myschema.text", false}, // not pg_catalog
		{"custom with pg_catalog-like prefix", "pg_custom.text", false},

		// Complex combinations
		{"pg_catalog uppercase array", "PG_CATALOG.TEXT[]", true},
		{"mixed case with pg_catalog and parentheses", "pg_catalog.VarChar(100)", true},
		{"uppercase with precision and array", "NUMERIC(10,2)[]", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBuiltInType(tt.typeName)
			if result != tt.expected {
				t.Errorf("IsBuiltInType(%q) = %v; want %v", tt.typeName, result, tt.expected)
			}
		})
	}
}

func TestIsTextLikeType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		expected bool
	}{
		// Text types
		{"text", "text", true},
		{"TEXT uppercase", "TEXT", true},
		{"Text mixed case", "Text", true},

		// Varchar types
		{"varchar", "varchar", true},
		{"varchar with length", "varchar(255)", true},
		{"VARCHAR uppercase", "VARCHAR", true},
		{"VarChar mixed case", "VarChar(100)", true},
		{"character varying", "character varying", true},
		{"character varying with length", "character varying(100)", true},
		{"CHARACTER VARYING uppercase", "CHARACTER VARYING", true},
		{"Character Varying mixed case", "Character Varying(50)", true},

		// Char types
		{"char", "char", true},
		{"char with length", "char(10)", true},
		{"CHAR uppercase", "CHAR", true},
		{"Char mixed case", "Char(5)", true},
		{"character", "character", true},
		{"character with length", "character(20)", true},
		{"CHARACTER uppercase", "CHARACTER", true},
		{"Character mixed case", "Character(15)", true},

		// Bpchar (internal name for char)
		{"bpchar", "bpchar", true},
		{"BPCHAR uppercase", "BPCHAR", true},

		// Non-text types
		{"integer", "integer", false},
		{"int", "int", false},
		{"bigint", "bigint", false},
		{"boolean", "boolean", false},
		{"uuid", "uuid", false},
		{"jsonb", "jsonb", false},
		{"timestamp", "timestamp", false},
		{"date", "date", false},
		{"numeric", "numeric", false},
		{"decimal", "decimal", false},

		// Custom types
		{"custom enum", "status_enum", false},
		{"custom type", "my_type", false},

		// Edge cases
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"partial match 1", "varchar_custom", false}, // contains varchar but not text-like
		{"partial match 2", "custom_text", false},     // contains text but not text-like
		{"partial match 3", "mychar", false},          // contains char but not text-like

		// Array types should not be text-like (function doesn't strip array suffix)
		{"text array", "text[]", false},
		{"varchar array", "varchar[]", false},
		{"char array", "char[]", false},

		// With pg_catalog prefix (function doesn't strip schema prefix)
		{"pg_catalog.text", "pg_catalog.text", false},
		{"pg_catalog.varchar", "pg_catalog.varchar", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTextLikeType(tt.typeName)
			if result != tt.expected {
				t.Errorf("IsTextLikeType(%q) = %v; want %v", tt.typeName, result, tt.expected)
			}
		})
	}
}
