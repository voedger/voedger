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
		if pk.Contains(k) {
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

// Returns all available privileges on specified type.
func allPrivilegesOnType(tk TypeKind) PrivilegeKinds {
	switch tk {
	case TypeKind_GRecord, TypeKind_GDoc,
		TypeKind_CRecord, TypeKind_CDoc,
		TypeKind_WRecord, TypeKind_WDoc,
		TypeKind_ORecord, TypeKind_ODoc,
		TypeKind_Object,
		TypeKind_ViewRecord:
		return PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select}
	case TypeKind_Command, TypeKind_Query:
		return PrivilegeKinds{PrivilegeKind_Execute}
	case TypeKind_Workspace:
		return PrivilegeKinds{PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute}
	case TypeKind_Role:
		return PrivilegeKinds{PrivilegeKind_Inherits}
	}
	return nil
}

// Validates names for privilege on. Returns sorted names without duplicates.
//
//   - If on is empty then returns error
//   - If on contains unknown name then returns error
//   - If on contains name of type that can not to be privileged then returns error
//   - If on contains names of mixed types then returns error.
func validatePrivilegeOnNames(app IAppDef, on ...QName) (QNames, error) {
	if len(on) == 0 {
		return nil, ErrPrivilegeOnMissed
	}

	names := QNamesFrom(on...)
	onKind := 0

	for _, n := range names {
		t := app.TypeByName(n)
		if t == nil {
			return nil, fmt.Errorf("%w: %v", ErrTypeNotFound, n)
		}
		k := onKind
		switch t.Kind() {
		case TypeKind_GRecord, TypeKind_GDoc,
			TypeKind_CRecord, TypeKind_CDoc,
			TypeKind_WRecord, TypeKind_WDoc,
			TypeKind_ORecord, TypeKind_ODoc,
			TypeKind_Object,
			TypeKind_ViewRecord:
			k = 1
		case TypeKind_Command, TypeKind_Query:
			k = 2
		case TypeKind_Workspace:
			k = 3
		case TypeKind_Role:
			k = 4
		default:
			return nil, fmt.Errorf("type can not to be privileged: %w: %v", ErrInvalidTypeKind, t)
		}

		if onKind != k {
			if onKind == 0 {
				onKind = k
			} else {
				return nil, fmt.Errorf("%w: %v and %v", ErrPrivilegeOnMixed, app.Type(names[0]), t)
			}
		}
	}

	return names, nil
}
