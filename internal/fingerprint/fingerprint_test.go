package fingerprint

import (
	"encoding/json"
	"testing"

	"github.com/pgschema/pgschema/ir"
)

func TestComputeFingerprint(t *testing.T) {
	// Create a simple IR for testing
	testIR := &ir.IR{
		Metadata: ir.Metadata{
			DatabaseVersion: "16.0",
		},
		Schemas: map[string]*ir.Schema{
			"public": {
				Name:       "public",
				Owner:      "postgres",
				Tables:     make(map[string]*ir.Table),
				Views:      make(map[string]*ir.View),
				Functions:  make(map[string]*ir.Function),
				Procedures: make(map[string]*ir.Procedure),
				Sequences:  make(map[string]*ir.Sequence),
				Types:      make(map[string]*ir.Type),
			},
		},
	}

	fingerprint, err := ComputeFingerprint(testIR, "public")
	if err != nil {
		t.Fatalf("ComputeFingerprint failed: %v", err)
	}

	// Check that fingerprint was computed
	if fingerprint.Hash == "" {
		t.Error("Fingerprint hash is empty")
	}

	// Just check that hash was computed - no other fields to verify
}

func TestComputeFingerprintWithTable(t *testing.T) {
	// Create IR with a table
	testIR := &ir.IR{
		Metadata: ir.Metadata{
			DatabaseVersion: "16.0",
		},
		Schemas: map[string]*ir.Schema{
			"public": {
				Name:  "public",
				Owner: "postgres",
				Tables: map[string]*ir.Table{
					"users": {
						Schema: "public",
						Name:   "users",
						Type:   ir.TableTypeBase,
						Columns: []*ir.Column{
							{
								Name:         "id",
								Position:     1,
								DataType:     "integer",
								IsNullable:   false,
								DefaultValue: nil,
							},
							{
								Name:         "name",
								Position:     2,
								DataType:     "text",
								IsNullable:   true,
								DefaultValue: nil,
							},
						},
						Constraints: make(map[string]*ir.Constraint),
						Indexes:     make(map[string]*ir.Index),
						Triggers:    make(map[string]*ir.Trigger),
						Policies:    make(map[string]*ir.RLSPolicy),
					},
				},
				Views:      make(map[string]*ir.View),
				Functions:  make(map[string]*ir.Function),
				Procedures: make(map[string]*ir.Procedure),
				Sequences:  make(map[string]*ir.Sequence),
				Types:      make(map[string]*ir.Type),
			},
		},
	}

	fingerprint, err := ComputeFingerprint(testIR, "public")
	if err != nil {
		t.Fatalf("ComputeFingerprint failed: %v", err)
	}

	// Just verify hash was computed for schema with table
	if fingerprint.Hash == "" {
		t.Error("Fingerprint hash is empty")
	}
}

func TestFingerprintConsistency(t *testing.T) {
	// Create the same IR twice and ensure fingerprints are identical
	createTestIR := func() *ir.IR {
		return &ir.IR{
			Metadata: ir.Metadata{
				DatabaseVersion: "16.0",
			},
			Schemas: map[string]*ir.Schema{
				"public": {
					Name:  "public",
					Owner: "postgres",
					Tables: map[string]*ir.Table{
						"test_table": {
							Schema: "public",
							Name:   "test_table",
							Type:   ir.TableTypeBase,
							Columns: []*ir.Column{
								{
									Name:         "id",
									Position:     1,
									DataType:     "integer",
									IsNullable:   false,
									DefaultValue: nil,
								},
							},
							Constraints: make(map[string]*ir.Constraint),
							Indexes:     make(map[string]*ir.Index),
							Triggers:    make(map[string]*ir.Trigger),
							Policies:    make(map[string]*ir.RLSPolicy),
						},
					},
					Views:      make(map[string]*ir.View),
					Functions:  make(map[string]*ir.Function),
					Procedures: make(map[string]*ir.Procedure),
					Sequences:  make(map[string]*ir.Sequence),
					Types:      make(map[string]*ir.Type),
				},
			},
		}
	}

	ir1 := createTestIR()
	ir2 := createTestIR()

	fingerprint1, err := ComputeFingerprint(ir1, "public")
	if err != nil {
		t.Fatalf("ComputeFingerprint failed for IR1: %v", err)
	}

	fingerprint2, err := ComputeFingerprint(ir2, "public")
	if err != nil {
		t.Fatalf("ComputeFingerprint failed for IR2: %v", err)
	}

	// Fingerprints should be identical (excluding timestamp)
	if fingerprint1.Hash != fingerprint2.Hash {
		t.Errorf("Fingerprint hashes differ:\nIR1: %s\nIR2: %s", fingerprint1.Hash, fingerprint2.Hash)
	}
}

func TestFingerprintSerialization(t *testing.T) {
	// Test that fingerprints can be properly serialized/deserialized
	testIR := &ir.IR{
		Metadata: ir.Metadata{
			DatabaseVersion: "16.0",
		},
		Schemas: map[string]*ir.Schema{
			"public": {
				Name:       "public",
				Owner:      "postgres",
				Tables:     make(map[string]*ir.Table),
				Views:      make(map[string]*ir.View),
				Functions:  make(map[string]*ir.Function),
				Procedures: make(map[string]*ir.Procedure),
				Sequences:  make(map[string]*ir.Sequence),
				Types:      make(map[string]*ir.Type),
			},
		},
	}

	originalFingerprint, err := ComputeFingerprint(testIR, "public")
	if err != nil {
		t.Fatalf("ComputeFingerprint failed: %v", err)
	}

	// Serialize to JSON
	data, err := json.Marshal(originalFingerprint)
	if err != nil {
		t.Fatalf("JSON marshaling failed: %v", err)
	}

	// Deserialize from JSON
	var deserializedFingerprint SchemaFingerprint
	err = json.Unmarshal(data, &deserializedFingerprint)
	if err != nil {
		t.Fatalf("JSON unmarshaling failed: %v", err)
	}

	// Check that hash matches
	if originalFingerprint.Hash != deserializedFingerprint.Hash {
		t.Errorf("Hash mismatch after serialization: %s != %s", originalFingerprint.Hash, deserializedFingerprint.Hash)
	}
}

func TestHashObject(t *testing.T) {
	// Test basic object hashing
	obj1 := map[string]interface{}{
		"name": "test",
		"type": "table",
	}

	obj2 := map[string]interface{}{
		"name": "test",
		"type": "table",
	}

	obj3 := map[string]interface{}{
		"name": "test2",
		"type": "table",
	}

	hash1, err := hashObject(obj1)
	if err != nil {
		t.Fatalf("hashObject failed for obj1: %v", err)
	}

	hash2, err := hashObject(obj2)
	if err != nil {
		t.Fatalf("hashObject failed for obj2: %v", err)
	}

	hash3, err := hashObject(obj3)
	if err != nil {
		t.Fatalf("hashObject failed for obj3: %v", err)
	}

	// Same objects should have same hash
	if hash1 != hash2 {
		t.Errorf("Identical objects have different hashes: %s != %s", hash1, hash2)
	}

	// Different objects should have different hashes
	if hash1 == hash3 {
		t.Errorf("Different objects have same hash: %s", hash1)
	}
}

