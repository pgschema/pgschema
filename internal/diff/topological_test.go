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
