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

// Makes PrivilegeKinds from specified kinds.
func PrivilegeKindsFrom(kinds ...PrivilegeKind) PrivilegeKinds {
	pk := make(PrivilegeKinds, 0, len(kinds))
	for _, k := range kinds {
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

// Returns all available privileges on specified type.
//
// If type can not to be privileged then returns empty slice.
func AllPrivilegesOnType(tk TypeKind) (pk PrivilegeKinds) {
	switch tk {
	case TypeKind_GRecord, TypeKind_GDoc,
		TypeKind_CRecord, TypeKind_CDoc,
		TypeKind_WRecord, TypeKind_WDoc,
		TypeKind_ORecord, TypeKind_ODoc,
		TypeKind_Object,
		TypeKind_ViewRecord:
		pk = append(pk, PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select)
	case TypeKind_Command, TypeKind_Query:
		pk = append(pk, PrivilegeKind_Execute)
	case TypeKind_Workspace:
		pk = append(pk, PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute)
	case TypeKind_Role:
		pk = append(pk, PrivilegeKind_Inherits)
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
func (pk PrivilegeKinds) String() string {
	var s string
	for i, k := range pk {
		if i > 0 {
			s = strings.Join([]string{s, k.TrimString()}, " ")
		} else {
			s = k.TrimString()
		}
	}
	return fmt.Sprintf("[%s]", s)
}

// Renders an PrivilegeKind in human-readable form, without "PrivilegeKind_" prefix,
// suitable for debugging or error messages
func (k PrivilegeKind) TrimString() string {
	const pref = "PrivilegeKind_"
	return strings.TrimPrefix(k.String(), pref)
}

// Validates names for privilege on. Returns sorted names without duplicates.
//
//   - If on is empty then returns error
//   - If on contains unknown name then returns error
//   - If on contains name of type that can not to be privileged then returns error
//   - If on contains names of mixed types then returns error.
func validatePrivilegeOnNames(app IAppDef, on ...QName) (QNames, error) {
	if len(on) == 0 {
		return nil, ErrMissed("privilege object names")
	}

	names := QNamesFrom(on...)
	onType := TypeKind_null

	for _, n := range names {
		t := app.TypeByName(n)
		if t == nil {
			return nil, ErrNotFound("type «%v»", n)
		}
		k := onType
		switch t.Kind() {
		case TypeKind_GRecord, TypeKind_GDoc,
			TypeKind_CRecord, TypeKind_CDoc,
			TypeKind_WRecord, TypeKind_WDoc,
			TypeKind_ORecord, TypeKind_ODoc,
			TypeKind_Object,
			TypeKind_ViewRecord:
			k = TypeKind_GRecord
		case TypeKind_Command, TypeKind_Query:
			k = TypeKind_Command
		case TypeKind_Workspace:
			k = TypeKind_Workspace
		case TypeKind_Role:
			k = TypeKind_Role
		default:
			return nil, fmt.Errorf("type can not to be privileged: %w: %v", ErrInvalidTypeKind, t)
		}

		if onType != k {
			if onType == TypeKind_null {
				onType = k
			} else {
				return nil, fmt.Errorf("%w: %v and %v", ErrPrivilegeOnMixed, app.Type(names[0]), t)
			}
		}
	}

	return names, nil
}
