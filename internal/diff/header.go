package diff

import (
	"fmt"
	"strings"

	"github.com/pgplex/pgschema/internal/version"
	"github.com/pgplex/pgschema/ir"
)

// GenerateDumpHeader generates the header for database dumps with metadata
func GenerateDumpHeader(schemaIR *ir.IR) string {
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
