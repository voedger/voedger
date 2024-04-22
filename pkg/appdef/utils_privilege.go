/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"slices"
	"strings"
)

func (k PrivilegeKind) TrimString() string {
	const pref = "PrivilegeKind_"
	return strings.TrimPrefix(k.String(), pref)
}

// Makes PrivilegeKinds from specified kinds.
func PrivilegeKindsFrom(kk ...PrivilegeKind) PrivilegeKinds {
	pk := make(PrivilegeKinds, 0, len(kk))
	for _, k := range kk {
		if k != PrivilegeKind_null {
			pk = append(pk, k)
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

func (kk PrivilegeKinds) String() string {
	s := ""
	for _, k := range kk {
		s = strings.Join([]string{s, k.TrimString()}, ", ")
	}
	return s
}
