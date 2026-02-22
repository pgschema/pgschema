package util

import (
	"bufio"
	"io"
	"os"
	"strings"
)

const schemaHeaderPrefix = "-- Dumped from schema: "

// DetectSchemaFromFile reads the header of a SQL dump file and extracts the
// schema name from the "-- Dumped from schema: <name>" metadata line.
// Returns empty string if the header is not found.
func DetectSchemaFromFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return detectSchemaFromReader(f)
}

// detectSchemaFromReader reads from an io.Reader and extracts the schema name
// from the pgschema dump header. Only scans the first 20 lines (header area).
func detectSchemaFromReader(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	for i := 0; i < 20 && scanner.Scan(); i++ {
		line := scanner.Text()
		if strings.HasPrefix(line, schemaHeaderPrefix) {
			return strings.TrimSpace(line[len(schemaHeaderPrefix):]), nil
		}
	}
	return "", scanner.Err()
}
