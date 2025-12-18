package diff

import (
	"fmt"
	"testing"

	"github.com/pgschema/pgschema/ir"
)

func TestTopologicallySortTablesHandlesCycles(t *testing.T) {
	tables := []*ir.Table{
		newTestTable("a"),
		newTestTable("b", "a"),
		newTestTable("c", "b"),
		newTestTable("x", "y"), // cycle x <-> y
		newTestTable("y", "x"),
		newTestTable("z", "y"), // depends on the cycle
	}

	sorted := topologicallySortTables(tables)
	if len(sorted) != len(tables) {
		t.Fatalf("expected %d tables, got %d", len(tables), len(sorted))
	}

	order := make(map[string]int, len(sorted))
	for idx, tbl := range sorted {
		order[tbl.Name] = idx
	}

	assertBefore := func(first, second string) {
		if order[first] >= order[second] {
			t.Fatalf("expected %s to appear before %s in %v", first, second, order)
		}
	}

	assertBefore("a", "b")
	assertBefore("b", "c")
	assertBefore("y", "z") // dependent tables still come afterwards

	// Cycle members should have a deterministic order (insertion order in this implementation)
	if order["x"] >= order["y"] {
		t.Fatalf("expected x to be ordered before y for deterministic output, got %v", order)
	}
}

func newTestTable(name string, deps ...string) *ir.Table {
	constraints := make(map[string]*ir.Constraint)
	for idx, dep := range deps {
		constraints[fmt.Sprintf("fk_%s_%d", name, idx)] = &ir.Constraint{
			Type:             ir.ConstraintTypeForeignKey,
			ReferencedSchema: "public",
			ReferencedTable:  dep,
		}
	}

	return &ir.Table{
		Schema:      "public",
		Name:        name,
		Constraints: constraints,
	}
}

func TestTopologicallySortTypesHandlesCycles(t *testing.T) {
	types := []*ir.Type{
		// Simple chain: a <- b <- c
		newTestEnumType("a"),
		newTestCompositeType("b", "a"),
		newTestCompositeType("c", "b"),
		// Cycle: x <-> y (theoretically impossible in PostgreSQL but test handles it)
		newTestCompositeType("x", "y"),
		newTestCompositeType("y", "x"),
		// Type depending on the cycle
		newTestCompositeType("z", "y"),
	}

	sorted := topologicallySortTypes(types)
	if len(sorted) != len(types) {
		t.Fatalf("expected %d types, got %d", len(types), len(sorted))
	}

	order := make(map[string]int, len(sorted))
	for idx, typ := range sorted {
		order[typ.Name] = idx
	}

	assertBefore := func(first, second string) {
		if order[first] >= order[second] {
			t.Fatalf("expected %s to appear before %s in %v", first, second, order)
		}
	}

	// Verify simple chain ordering
	assertBefore("a", "b")
	assertBefore("b", "c")
	// Dependent types should still come after cycle members
	assertBefore("y", "z")

	// Cycle members should have a deterministic order (insertion order)
	if order["x"] >= order["y"] {
		t.Fatalf("expected x to be ordered before y for deterministic output, got %v", order)
	}
}

func TestTopologicallySortTypesMultipleNoDependencies(t *testing.T) {
	types := []*ir.Type{
		newTestEnumType("z"),
		newTestEnumType("a"),
		newTestEnumType("m"),
		newTestEnumType("b"),
	}

	sorted := topologicallySortTypes(types)
	if len(sorted) != len(types) {
		t.Fatalf("expected %d types, got %d", len(types), len(sorted))
	}

	// With no dependencies, should maintain deterministic alphabetical order
	order := make(map[string]int, len(sorted))
	for idx, typ := range sorted {
		order[typ.Name] = idx
	}

	// Verify deterministic ordering: a < b < m < z
	if order["a"] >= order["b"] || order["b"] >= order["m"] || order["m"] >= order["z"] {
		t.Fatalf("expected alphabetical order for types with no dependencies, got %v", order)
	}
}

func TestTopologicallySortTypesDomainReferencingCustomType(t *testing.T) {
	types := []*ir.Type{
		newTestEnumType("status_type"),
		newTestDomainType("status_domain", "status_type"),
		newTestCompositeType("person", "status_domain"),
	}

	sorted := topologicallySortTypes(types)
	if len(sorted) != len(types) {
		t.Fatalf("expected %d types, got %d", len(types), len(sorted))
	}

	order := make(map[string]int, len(sorted))
	for idx, typ := range sorted {
		order[typ.Name] = idx
	}

	assertBefore := func(first, second string) {
		if order[first] >= order[second] {
			t.Fatalf("expected %s to appear before %s in %v", first, second, order)
		}
	}

	// Verify correct dependency chain
	assertBefore("status_type", "status_domain")
	assertBefore("status_domain", "person")
}

func TestTopologicallySortTypesCompositeWithMultipleDependencies(t *testing.T) {
	types := []*ir.Type{
		newTestEnumType("status"),
		newTestEnumType("priority"),
		newTestEnumType("category"),
		newTestCompositeType("task", "status", "priority", "category"),
		newTestCompositeType("project", "task"),
	}

	sorted := topologicallySortTypes(types)
	if len(sorted) != len(types) {
		t.Fatalf("expected %d types, got %d", len(types), len(sorted))
	}

	order := make(map[string]int, len(sorted))
	for idx, typ := range sorted {
		order[typ.Name] = idx
	}

	assertBefore := func(first, second string) {
		if order[first] >= order[second] {
			t.Fatalf("expected %s to appear before %s in %v", first, second, order)
		}
	}

	// All dependencies should come before task
	assertBefore("status", "task")
	assertBefore("priority", "task")
	assertBefore("category", "task")
	// And task should come before project
	assertBefore("task", "project")
}

func newTestEnumType(name string) *ir.Type {
	return &ir.Type{
		Schema:     "public",
		Name:       name,
		Kind:       ir.TypeKindEnum,
		EnumValues: []string{"value1", "value2"},
	}
}

func newTestCompositeType(name string, deps ...string) *ir.Type {
	columns := make([]*ir.TypeColumn, len(deps))
	for idx, dep := range deps {
		columns[idx] = &ir.TypeColumn{
			Name:     fmt.Sprintf("col_%d", idx),
			DataType: dep, // References the type
			Position: idx + 1,
		}
	}

	return &ir.Type{
		Schema:  "public",
		Name:    name,
		Kind:    ir.TypeKindComposite,
		Columns: columns,
	}
}

func newTestDomainType(name, baseType string) *ir.Type {
	return &ir.Type{
		Schema:   "public",
		Name:     name,
		Kind:     ir.TypeKindDomain,
		BaseType: baseType,
	}
}
