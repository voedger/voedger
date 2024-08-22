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

// Returns "grant" if policy is allow, "revoke" if deny
func (p PolicyKind) ActionString() string {
	switch p {
	case PolicyKind_Allow:
		return "grant"
	case PolicyKind_Deny:
		return "revoke"
	}
	return p.TrimString()
}

// Renders an PolicyKind in human-readable form, without "PolicyKind_" prefix,
// suitable for debugging or error messages
func (p PolicyKind) TrimString() string {
	const pref = "PolicyKind_"
	return strings.TrimPrefix(p.String(), pref)
}

// Returns all available operations on specified type.
//
// If type can not to be used as resource for ACL rule then returns empty slice.
func allACLOperationsOnType(t IType) (ops set.Set[OperationKind]) {
	switch t.Kind() {
	case TypeKind_Any:
		switch t.QName() {
		case QNameANY:
			ops = set.From(OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute, OperationKind_Inherits)
		case QNameAnyStructure, QNameAnyRecord,
			QNameAnyGDoc, QNameAnyCDoc, QNameAnyWDoc,
			QNameAnySingleton,
			QNameAnyView:
			ops = set.From(OperationKind_Insert, OperationKind_Update, OperationKind_Select)
		case QNameAnyFunction, QNameAnyCommand, QNameAnyQuery:
			ops = set.From(OperationKind_Execute)
		}
	case TypeKind_GRecord, TypeKind_GDoc,
		TypeKind_CRecord, TypeKind_CDoc,
		TypeKind_WRecord, TypeKind_WDoc,
		TypeKind_ORecord, TypeKind_ODoc,
		TypeKind_Object,
		TypeKind_ViewRecord:
		ops = set.From(OperationKind_Insert, OperationKind_Update, OperationKind_Select)
	case TypeKind_Command, TypeKind_Query:
		ops = set.From(OperationKind_Execute)
	case TypeKind_Workspace:
		ops = set.From(OperationKind_Insert, OperationKind_Update, OperationKind_Select, OperationKind_Execute)
	case TypeKind_Role:
		ops = set.From(OperationKind_Inherits)
	}
	return ops
}

// Renders an OperationKind in human-readable form, without "OperationKind_" prefix,
// suitable for debugging or error messages
func (k OperationKind) TrimString() string {
	const pref = "OperationKind_"
	return strings.TrimPrefix(k.String(), pref)
}

// Validates specified field names by types. Returns error if any field is not found.
//
// If types contains any substitution then all fields are allowed.
func validateFieldNamesByTypes(tt IWithTypes, types []QName, fields []FieldName) (err error) {
	names := QNamesFrom(types...)

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

// Validates resource names for ACL. Returns sorted names without duplicates.
//
//   - If names is empty then returns error
//   - If names contains unknown name then returns error
//   - If names contains name of type that can not to be in ACL then returns error
//   - If names contains names of mixed types then returns error.
func validateACLResourceNames(tt IWithTypes, names ...QName) (QNames, error) {
	if len(names) == 0 {
		return nil, ErrMissed("privilege object names")
	}

	nn := QNamesFrom(names...)
	onType := TypeKind_null

	for _, n := range nn {
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
				return nil, ErrIncompatible("substitution «%v» can not be used in ACL", t)
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
			return nil, ErrIncompatible("type «%v» can not to be restricted by ACL", t)
		}

		if onType != k {
			if onType == TypeKind_null {
				onType = k
			} else {
				return nil, ErrIncompatible("incompatible resource types can not to be mixed in one ACL (%v and %v)", tt.Type(nn[0]), t)
			}
		}
	}

	return nn, nil
}
