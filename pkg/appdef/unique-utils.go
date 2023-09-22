/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
// For single field unique, the concatenation of `Unique` word and the field name is used.
// E.g., for type with single field «eMail» name "UniqueEMail" will returned.
//
// For multiply fields unique, the concatenation of the `Unique` word and two digits is used, e.g. "Unique01".
func generateUniqueName(u IUniques, fields []string) string {
	const pref = "Unique"
	if len(fields) == 1 {
		s := pref + cases.Title(language.English, cases.NoLower).String(fields[0])
		if u.UniqueByName(s) == nil {
			return s
		}
	}
	const tryCnt = MaxDefUniqueCount
	for i := 1; i < tryCnt; i++ {
		s := pref + fmt.Sprintf("%02d", i)
		if u.UniqueByName(s) == nil {
			return s
		}
	}
	panic(fmt.Errorf("unable to generate unique name for type: %w", ErrTooManyUniques))
}
