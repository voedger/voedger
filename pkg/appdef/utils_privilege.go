/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"fmt"
	"slices"
	"strings"
)

// Renders an PrivilegeKind in human-readable form, without "PrivilegeKind_" prefix,
// suitable for debugging or error messages
func (k PrivilegeKind) TrimString() string {
	const pref = "PrivilegeKind_"
	return strings.TrimPrefix(k.String(), pref)
}

// Makes PrivilegeKinds from specified kinds.
func PrivilegeKindsFrom(kk ...PrivilegeKind) PrivilegeKinds {
	pk := make(PrivilegeKinds, 0, len(kk))
	for _, k := range kk {
		if (k > PrivilegeKind_null) && (k < PrivilegeKind_count) {
			if !slices.Contains(pk, k) {
				pk = append(pk, k)
			}
		} else {
			panic(fmt.Errorf("%w: %v", ErrInvalidPrivilegeKind, k))
		}
	}
	return pk
}

// Returns is kinds contains the specified kind.
func (pk PrivilegeKinds) Contains(k PrivilegeKind) bool { return slices.Contains(pk, k) }

// Returns is kinds contains all specified kind.
func (pk PrivilegeKinds) ContainsAll(kk ...PrivilegeKind) bool {
	for _, k := range kk {
		if !pk.Contains(k) {
			return false
		}
	}
	return true
}

// Returns is kinds contains any from specified kind.
//
// If no kind specified then returns true.
func (pk PrivilegeKinds) ContainsAny(kk ...PrivilegeKind) bool {
	for _, k := range kk {
		if !pk.Contains(k) {
			return true
		}
	}
	return len(kk) == 0
}

// Renders an PrivilegeKinds in human-readable form, without "PrivilegeKind_" prefix,
// suitable for debugging or error messages
func (kk PrivilegeKinds) String() string {
	var s string
	for i, k := range kk {
		if i > 0 {
			s = strings.Join([]string{s, k.TrimString()}, " ")
		} else {
			s = k.TrimString()
		}
	}
	return fmt.Sprintf("[%s]", s)
}
