package fingerprint

import (
	"fmt"
)

// Compare compares two schema fingerprints and returns an error if they don't match
func Compare(expected, actual *SchemaFingerprint) error {
	if expected.Hash == actual.Hash {
		return nil // Fingerprints match
	}
	
	expectedPreview := expected.Hash
	if len(expectedPreview) > 16 {
		expectedPreview = expectedPreview[:16]
	}
	
	actualPreview := actual.Hash
	if len(actualPreview) > 16 {
		actualPreview = actualPreview[:16]
	}
	
	return fmt.Errorf("schema fingerprint mismatch - expected: %s, actual: %s", 
		expectedPreview, actualPreview)
}

