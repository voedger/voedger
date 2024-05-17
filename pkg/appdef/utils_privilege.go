/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"strings"

	"github.com/voedger/voedger/pkg/goutils/set"
)

// Returns "grant" if grant is true, otherwise "revoke".
func PrivilegeAccessControlString(grant bool) string {
	var result = []string{"grant", "revoke"}
	if grant {
		return result[0]
	}
	return result[1]
}

// Returns all available privileges on specified type.
//
// If type can not to be privileged then returns empty slice.
func allPrivilegesOnType(t IType) (pk set.Set[PrivilegeKind]) {
	switch t.Kind() {
	case TypeKind_Any:
		switch t.QName() {
		case QNameANY:
			pk = set.From(PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute, PrivilegeKind_Inherits)
		case QNameAnyStructure, QNameAnyRecord,
			QNameAnyGDoc, QNameAnyCDoc, QNameAnyWDoc,
			QNameAnySingleton,
			QNameAnyView:
			pk = set.From(PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select)
		case QNameAnyFunction, QNameAnyCommand, QNameAnyQuery:
			pk = set.From(PrivilegeKind_Execute)
		}
	case TypeKind_GRecord, TypeKind_GDoc,
		TypeKind_CRecord, TypeKind_CDoc,
		TypeKind_WRecord, TypeKind_WDoc,
		TypeKind_ORecord, TypeKind_ODoc,
		TypeKind_Object,
		TypeKind_ViewRecord:
		pk = set.From(PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select)
	case TypeKind_Command, TypeKind_Query:
		pk = set.From(PrivilegeKind_Execute)
	case TypeKind_Workspace:
		pk = set.From(PrivilegeKind_Insert, PrivilegeKind_Update, PrivilegeKind_Select, PrivilegeKind_Execute)
	case TypeKind_Role:
		pk = set.From(PrivilegeKind_Inherits)
	}
	return pk
}

// Renders an PrivilegeKind in human-readable form, without "PrivilegeKind_" prefix,
// suitable for debugging or error messages
func (k PrivilegeKind) TrimString() string {
	const pref = "PrivilegeKind_"
	return strings.TrimPrefix(k.String(), pref)
}

// Validates privilege on field names. Returns error if any field is not found.
//
// If on contains any substitution then all fields are allowed.
func validatePrivilegeOnFieldNames(tt IWithTypes, on []QName, fields []FieldName) (err error) {
	names := QNamesFrom(on...)

	allFields := map[FieldName]struct{}{}

	for _, n := range names {
		t := tt.Type(n)
		switch t.Kind() {
		case TypeKind_Any:
			// any subst allow to use any fields
			return nil
		case TypeKind_GRecord, TypeKind_GDoc,
			TypeKind_CRecord, TypeKind_CDoc,
			TypeKind_WRecord, TypeKind_WDoc,
			TypeKind_ORecord, TypeKind_ODoc,
			TypeKind_Object,
			TypeKind_ViewRecord:
			if ff, ok := t.(IFields); ok {
				for _, f := range ff.Fields() {
					allFields[f.Name()] = struct{}{}
				}
			}
		}
	}

	for _, f := range fields {
		if _, ok := allFields[f]; !ok {
			err = errors.Join(err, ErrFieldNotFound(f))
		}
	}

	return err
}

// Validates names for privilege on. Returns sorted names without duplicates.
//
//   - If on is empty then returns error
//   - If on contains unknown name then returns error
//   - If on contains name of type that can not to be privileged then returns error
//   - If on contains names of mixed types then returns error.
func validatePrivilegeOnNames(tt IWithTypes, on ...QName) (QNames, error) {
	if len(on) == 0 {
		return nil, ErrMissed("privilege object names")
	}

	names := QNamesFrom(on...)
	onType := TypeKind_null

	for _, n := range names {
		t := tt.TypeByName(n)
		if t == nil {
			return nil, ErrTypeNotFound(n)
		}
		k := onType
		switch t.Kind() {
		case TypeKind_Any:
			switch n {
			case QNameANY:
				k = TypeKind_Any
			case QNameAnyStructure, QNameAnyRecord,
				QNameAnyGDoc, QNameAnyCDoc, QNameAnyWDoc,
				QNameAnySingleton,
				QNameAnyView:
				k = TypeKind_GRecord
			case QNameAnyFunction, QNameAnyCommand, QNameAnyQuery:
				k = TypeKind_Command
			default:
				return nil, ErrIncompatible("substitution «%v» can not to be privileged", t)
			}
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
			return nil, ErrIncompatible("type «%v» can not to be privileged", t)
		}

		if onType != k {
			if onType == TypeKind_null {
				onType = k
			} else {
				return nil, ErrIncompatible("privileged object types mixed in list (%v and %v)", tt.Type(names[0]), t)
			}
		}
	}

	return names, nil
}
