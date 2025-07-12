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
