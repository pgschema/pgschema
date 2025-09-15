package ignore

import (
	"os"

	"github.com/BurntSushi/toml"
)

const (
	// IgnoreFileName is the default name of the ignore file
	IgnoreFileName = ".pgschemaignore"
)

// LoadIgnoreFile loads the .pgschemaignore file from the current directory
// Returns nil if the file doesn't exist (ignore functionality is optional)
func LoadIgnoreFile() (*IgnoreConfig, error) {
	return LoadIgnoreFileFromPath(IgnoreFileName)
}

// LoadIgnoreFileFromPath loads an ignore file from the specified path
// Returns nil if the file doesn't exist (ignore functionality is optional)
// Uses the structured TOML format internally
func LoadIgnoreFileFromPath(filePath string) (*IgnoreConfig, error) {
	return LoadIgnoreFileWithStructureFromPath(filePath)
}

// TomlConfig represents the TOML structure of the .pgschemaignore file
// This is used for parsing more complex configurations if needed in the future
type TomlConfig struct {
	Tables     TableIgnoreConfig     `toml:"tables,omitempty"`
	Views      ViewIgnoreConfig      `toml:"views,omitempty"`
	Functions  FunctionIgnoreConfig  `toml:"functions,omitempty"`
	Procedures ProcedureIgnoreConfig `toml:"procedures,omitempty"`
	Types      TypeIgnoreConfig      `toml:"types,omitempty"`
	Sequences  SequenceIgnoreConfig  `toml:"sequences,omitempty"`
}

// TableIgnoreConfig represents table-specific ignore configuration
type TableIgnoreConfig struct {
	Patterns []string `toml:"patterns,omitempty"`
}

// ViewIgnoreConfig represents view-specific ignore configuration
type ViewIgnoreConfig struct {
	Patterns []string `toml:"patterns,omitempty"`
}

// FunctionIgnoreConfig represents function-specific ignore configuration
type FunctionIgnoreConfig struct {
	Patterns []string `toml:"patterns,omitempty"`
}

// ProcedureIgnoreConfig represents procedure-specific ignore configuration
type ProcedureIgnoreConfig struct {
	Patterns []string `toml:"patterns,omitempty"`
}

// TypeIgnoreConfig represents type-specific ignore configuration
type TypeIgnoreConfig struct {
	Patterns []string `toml:"patterns,omitempty"`
}

// SequenceIgnoreConfig represents sequence-specific ignore configuration
type SequenceIgnoreConfig struct {
	Patterns []string `toml:"patterns,omitempty"`
}

// LoadIgnoreFileWithStructure loads the .pgschemaignore file using the structured TOML format
// and converts it to the simple IgnoreConfig structure
func LoadIgnoreFileWithStructure() (*IgnoreConfig, error) {
	return LoadIgnoreFileWithStructureFromPath(IgnoreFileName)
}

// LoadIgnoreFileWithStructureFromPath loads an ignore file using structured format from the specified path
func LoadIgnoreFileWithStructureFromPath(filePath string) (*IgnoreConfig, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist, return nil config (no filtering)
		return nil, nil
	} else if err != nil {
		// Other error accessing file
		return nil, err
	}

	// File exists, parse it
	var tomlConfig TomlConfig
	if _, err := toml.DecodeFile(filePath, &tomlConfig); err != nil {
		return nil, err
	}

	// Convert to simple IgnoreConfig structure
	config := &IgnoreConfig{
		Tables:     tomlConfig.Tables.Patterns,
		Views:      tomlConfig.Views.Patterns,
		Functions:  tomlConfig.Functions.Patterns,
		Procedures: tomlConfig.Procedures.Patterns,
		Types:      tomlConfig.Types.Patterns,
		Sequences:  tomlConfig.Sequences.Patterns,
	}

	return config, nil
}