package util

import (
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

func TestMapToEmbeddedPostgresVersion(t *testing.T) {
	tests := []struct {
		name          string
		majorVersion  int
		expected      embeddedpostgres.PostgresVersion
		expectError   bool
	}{
		{
			name:         "PostgreSQL 14",
			majorVersion: 14,
			expected:     embeddedpostgres.PostgresVersion("14.18.0"),
			expectError:  false,
		},
		{
			name:         "PostgreSQL 15",
			majorVersion: 15,
			expected:     embeddedpostgres.PostgresVersion("15.13.0"),
			expectError:  false,
		},
		{
			name:         "PostgreSQL 16",
			majorVersion: 16,
			expected:     embeddedpostgres.PostgresVersion("16.9.0"),
			expectError:  false,
		},
		{
			name:         "PostgreSQL 17",
			majorVersion: 17,
			expected:     embeddedpostgres.PostgresVersion("17.5.0"),
			expectError:  false,
		},
		{
			name:         "Unsupported version 13",
			majorVersion: 13,
			expected:     "",
			expectError:  true,
		},
		{
			name:         "Unsupported version 18",
			majorVersion: 18,
			expected:     "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mapToEmbeddedPostgresVersion(tt.majorVersion)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected version %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestParseVersionString(t *testing.T) {
	tests := []struct {
		name        string
		versionStr  string
		expected    int
		expectError bool
	}{
		{
			name:        "Simple version 17.5",
			versionStr:  "17.5",
			expected:    17,
			expectError: false,
		},
		{
			name:        "Version with patch 17.5.0",
			versionStr:  "17.5.0",
			expected:    17,
			expectError: false,
		},
		{
			name:        "Version with prefix",
			versionStr:  "PostgreSQL 17.5",
			expected:    17,
			expectError: false,
		},
		{
			name:        "Version 14.18.0",
			versionStr:  "14.18.0",
			expected:    14,
			expectError: false,
		},
		{
			name:        "Version with whitespace",
			versionStr:  "  16.9  ",
			expected:    16,
			expectError: false,
		},
		{
			name:        "Invalid version",
			versionStr:  "invalid",
			expected:    0,
			expectError: true,
		},
		{
			name:        "Empty string",
			versionStr:  "",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseVersionString(tt.versionStr)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected version %d, got %d", tt.expected, result)
				}
			}
		})
	}
}
