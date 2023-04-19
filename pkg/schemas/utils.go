/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import (
	"fmt"
	"strings"

	"github.com/voedger/voedger/pkg/istructs"
)

// Returns is fixed width data kind
func IsFixedWidthDataKind(kind DataKind) bool {
	switch kind {
	case
		istructs.DataKind_int32,
		istructs.DataKind_int64,
		istructs.DataKind_float32,
		istructs.DataKind_float64,
		istructs.DataKind_QName,
		istructs.DataKind_bool,
		istructs.DataKind_RecordID:
		return true
	}
	return false
}

// Returns is container system
func IsSysContainer(n string) bool {
	return strings.HasPrefix(n, istructs.SystemFieldPrefix) && // fast check
		// then more accuracy
		((n == istructs.SystemContainer_ViewPartitionKey) ||
			(n == istructs.SystemContainer_ViewClusteringCols) ||
			(n == istructs.SystemContainer_ViewValue))
}

// Returns is string is valid identifier and error if not
func ValidIdent(ident string) (bool, error) {
	if len(ident) < 1 {
		return false, ErrNameMissed
	}

	if len(ident) > MaxIdentLen {
		return false, fmt.Errorf("ident too long: %w", ErrInvalidName)
	}

	const (
		char_a rune = 97
		char_A rune = 65
		char_z rune = 122
		char_Z rune = 90
		char_0 rune = 48
		char_9 rune = 57
		char__ rune = 95
	)

	digit := func(r rune) bool {
		return (char_0 <= r) && (r <= char_9)
	}

	letter := func(r rune) bool {
		return ((char_a <= r) && (r <= char_z)) || ((char_A <= r) && (r <= char_Z))
	}

	underScore := func(r rune) bool {
		return r == char__
	}

	for p, c := range ident {
		if !letter(c) && !underScore(c) {
			if (p == 0) || !digit(c) {
				return false, fmt.Errorf("name char «%c» at pos %d is not valid: %w", c, p, ErrInvalidName)
			}
		}
	}

	return true, nil
}

// Returns has qName valid package and entity identifiers and error if not
func ValidQName(qName QName) (bool, error) {
	if qName == istructs.NullQName {
		return true, nil
	}
	if ok, err := ValidIdent(qName.Pkg()); !ok {
		return false, err
	}
	if ok, err := ValidIdent(qName.Entity()); !ok {
		return false, err
	}
	return true, nil
}

// Returns is field system
func IsSysField(n string) bool {
	return strings.HasPrefix(n, istructs.SystemFieldPrefix) && // fast check
		// then more accuracy
		((n == istructs.SystemField_QName) ||
			(n == istructs.SystemField_ID) ||
			(n == istructs.SystemField_ParentID) ||
			(n == istructs.SystemField_Container) ||
			(n == istructs.SystemField_IsActive))
}
