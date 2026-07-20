package collections

import (
	"cmp"
	"slices"
)

// RemoveDuplicates removes duplicates from the slice and returns a new slice with unique values.
func RemoveDuplicates[T cmp.Ordered](values []T) []T {
	keys := make(map[T]struct{})
	var uniqueValues []T
	for _, value := range values {
		if _, ok := keys[value]; !ok {
			keys[value] = struct{}{}
			uniqueValues = append(uniqueValues, value)
		}
	}
	slices.Sort(uniqueValues)
	return uniqueValues
}
