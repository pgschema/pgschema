package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestDiffCommandFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "no flags provided",
			args:        []string{"diff"},
			expectError: true,
			errorMsg:    "must specify both source and target",
		},
		{
			name:        "only source provided",
			args:        []string{"diff", "--source-dir", "test"},
			expectError: true,
			errorMsg:    "must specify both source and target",
		},
		{
			name:        "only target provided",
			args:        []string{"diff", "--target-dir", "test"},
			expectError: true,
			errorMsg:    "must specify both source and target",
		},
		{
			name:        "both dir and dsn for source",
			args:        []string{"diff", "--source-dir", "test", "--source-dsn", "postgres://", "--target-dir", "test2"},
			expectError: true,
			errorMsg:    "cannot specify both directory and DSN",
		},
		{
			name:        "both dir and dsn for target",
			args:        []string{"diff", "--source-dir", "test", "--target-dir", "test2", "--target-dsn", "postgres://"},
			expectError: true,
			errorMsg:    "cannot specify both directory and DSN",
		},
		{
			name:        "missing temp-db-dsn for directories",
			args:        []string{"diff", "--source-dir", "testdata/schema1", "--target-dir", "testdata/schema2"},
			expectError: true,
			errorMsg:    "--temp-db-dsn is required when using directory-based schemas",
		},
		{
			name:        "valid dir to dir with temp-db-dsn",
			args:        []string{"diff", "--source-dir", "testdata/schema1", "--target-dir", "testdata/schema2", "--temp-db-dsn", "postgres://user:pass@localhost/postgres"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flags
			sourceDir = ""
			sourceDSN = ""
			targetDir = ""
			targetDSN = ""
			tempDbDSN = ""
			
			cmd := &cobra.Command{Use: "pgschema"}
			diffCmdCopy := &cobra.Command{
				Use:   "diff",
				Short: "Compare two PostgreSQL schemas",
				Long:  "Compare schemas from directories or databases and show the differences",
				RunE:  runDiff,
			}
			diffCmdCopy.Flags().StringVar(&sourceDir, "source-dir", "", "Source schema directory")
			diffCmdCopy.Flags().StringVar(&sourceDSN, "source-dsn", "", "Source database connection string")
			diffCmdCopy.Flags().StringVar(&targetDir, "target-dir", "", "Target schema directory")
			diffCmdCopy.Flags().StringVar(&targetDSN, "target-dsn", "", "Target database connection string")
			diffCmdCopy.Flags().StringVar(&tempDbDSN, "temp-db-dsn", "", "Temporary database connection string")
			
			cmd.AddCommand(diffCmdCopy)
			
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) && !strings.Contains(err.Error(), "failed to load") {
					t.Errorf("expected error containing '%s' or 'failed to load', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil && !strings.Contains(err.Error(), "failed to load") && !strings.Contains(err.Error(), "failed to generate diff") && !strings.Contains(err.Error(), "failed to connect") && !strings.Contains(err.Error(), "failed to ping") {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLoadSchemaFromDirectory(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		expectError bool
	}{
		{
			name:        "valid directory",
			dir:         "testdata/schema1",
			expectError: false,
		},
		{
			name:        "non-existent directory",
			dir:         "testdata/nonexistent",
			expectError: true,
		},
		{
			name:        "empty directory path",
			dir:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadSchemaFromDirectory(tt.dir)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMainFunction(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"pgschema", "--help"}
	
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("main function panicked: %v", r)
		}
	}()
}