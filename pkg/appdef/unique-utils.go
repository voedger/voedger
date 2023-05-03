/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"strconv"
)

// If the slices have duplicates, then the indices of the first pair are returned, otherwise (-1, -1)
func duplicates[T comparable](s []T) (int, int) {
	for i := range s {
		for j := i + 1; j < len(s); j++ {
			if s[i] == s[j] {
				return i, j
			}
		}
	}
	return -1, -1
}

// Returns is slice sub is a subset of slice set, i.e. all elements from sub exist in set
func subSet[T comparable](sub, set []T) bool {
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
func overlaps[T comparable](set1, set2 []T) bool {
	return subSet(set1, set2) || subSet(set2, set1)
}

// Generates name for unique.
//
// For single field unique, the concationation of the definition name, the `Unique` word and the field name is used.
func generateUniqueName(def *def, fields []string) string {
	pref := fmt.Sprintf("%sUnique", def.QName().Entity())
	if len(fields) == 1 {
		s := pref + fields[0]
		if def.Unique(s) == nil {
			return s
		}
	}
	const tryCnt = 100
	for i := 0; i < tryCnt; i++ {
		s := pref + strconv.Itoa(i)
		if def.Unique(s) == nil {
			return s
		}
	}
	panic(fmt.Errorf("%v: unable to generate unique name for definition: %w", def.QName(), ErrNameMissed))
}
