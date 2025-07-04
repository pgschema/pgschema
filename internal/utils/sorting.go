package utils

import (
	"sort"
)

// SortedKeys returns sorted keys from a map[string]T
func SortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// SortEntitiesByName sorts a slice of entities by their Name field
func SortEntitiesByName[T interface{ GetName() string }](entities []T) []T {
	sorted := make([]T, len(entities))
	copy(sorted, entities)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].GetName() < sorted[j].GetName()
	})
	return sorted
}

// SortEntitiesBySchema sorts a slice of entities by their Schema.Name key
func SortEntitiesBySchema[T interface{ GetSchemaName() string }](entities []T) []T {
	sorted := make([]T, len(entities))
	copy(sorted, entities)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].GetSchemaName() < sorted[j].GetSchemaName()
	})
	return sorted
}

// SortEntitiesBySchemaAndName sorts entities by schema first, then by name
func SortEntitiesBySchemaAndName[T interface {
	GetSchemaName() string
	GetName() string
}](entities []T) []T {
	sorted := make([]T, len(entities))
	copy(sorted, entities)
	sort.Slice(sorted, func(i, j int) bool {
		schemaI := sorted[i].GetSchemaName()
		schemaJ := sorted[j].GetSchemaName()
		if schemaI != schemaJ {
			return schemaI < schemaJ
		}
		return sorted[i].GetName() < sorted[j].GetName()
	})
	return sorted
}

// SortEntitiesByKey sorts entities by a custom key function
func SortEntitiesByKey[T any](entities []T, keyFunc func(T) string) []T {
	sorted := make([]T, len(entities))
	copy(sorted, entities)
	sort.Slice(sorted, func(i, j int) bool {
		return keyFunc(sorted[i]) < keyFunc(sorted[j])
	})
	return sorted
}

// SortEntitiesBy sorts entities using a custom comparison function
func SortEntitiesBy[T any](entities []T, less func(i, j T) bool) []T {
	sorted := make([]T, len(entities))
	copy(sorted, entities)
	sort.Slice(sorted, func(i, j int) bool {
		return less(sorted[i], sorted[j])
	})
	return sorted
}