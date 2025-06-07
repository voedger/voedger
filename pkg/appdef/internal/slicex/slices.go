/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package slicex

import "sort"

// If the slices have duplicates, then the indices of the first pair are returned, otherwise (-1, -1)
func FindDuplicates[T comparable](s []T) (int, int) {
	for i := range s {
		for j := i + 1; j < len(s); j++ {
			if s[i] == s[j] {
				return i, j
			}
		}
	}
	return -1, -1
}

// Inserts element v into slice s in sorted order using comp function.
//
// If v already exists in s, then it is replaced with v, not added
func InsertInSort[T any, S ~[]T](s S, v T, comp func(T, T) int) S {
	i := sort.Search(len(s), func(i int) bool { return comp(s[i], v) >= 0 })
	if (i >= len(s)) || (comp(s[i], v) != 0) {
		s = append(s, v)
		copy(s[i+1:], s[i:])
	}
	s[i] = v
	return s
}

// Returns is slice sub is a subset of slice set, i.e. all elements from sub exist in set
func IsSubSet[T comparable](sub, set []T) bool {
	for _, v1 := range sub {
		found := false
		for _, v2 := range set {
			found = v1 == v2
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Returns is set1 and set2 overlaps, i.e. set1 is subset of set2 or set2 is subset of set1
func Overlaps[T comparable](set1, set2 []T) bool {
	return IsSubSet(set1, set2) || IsSubSet(set2, set1)
}
