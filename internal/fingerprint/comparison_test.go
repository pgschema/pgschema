package fingerprint

import (
	"strings"
	"testing"
)

func TestCompare_IdenticalFingerprints(t *testing.T) {
	// Create identical fingerprints
	fingerprint1 := &SchemaFingerprint{Hash: "same_hash_12345"}
	fingerprint2 := &SchemaFingerprint{Hash: "same_hash_12345"}

	err := Compare(fingerprint1, fingerprint2)

	if err != nil {
		t.Errorf("Identical fingerprints should match, got error: %v", err)
	}
}

func TestCompare_DifferentFingerprints(t *testing.T) {
	// Create fingerprints with different hashes
	fingerprint1 := &SchemaFingerprint{Hash: "hash_12345"}
	fingerprint2 := &SchemaFingerprint{Hash: "hash_67890"}

	err := Compare(fingerprint1, fingerprint2)

	if err == nil {
		t.Error("Different fingerprints should not match")
	}

	// Check error message format
	expectedSubstrings := []string{"schema fingerprint mismatch", "hash_1234", "hash_6789"}
	for _, substring := range expectedSubstrings {
		if !strings.Contains(err.Error(), substring) {
			t.Errorf("Error message should contain '%s', got: %s", substring, err.Error())
		}
	}
}

