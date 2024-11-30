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

// Returns iterator over ACL rules.
//
// ACL Rules are visited in the order they are added.
func ACL(acl IWithACL) func(func(IACLRule) bool) {
	return acl.ACL
}

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
// If type can not to be used for ACL rule then returns empty slice.
func allACLOperationsOnType(t IType) (ops set.Set[OperationKind]) {
	switch t.Kind() {
	case TypeKind_GRecord, TypeKind_GDoc,
		TypeKind_CRecord, TypeKind_CDoc,
		TypeKind_WRecord, TypeKind_WDoc,
		TypeKind_ORecord, TypeKind_ODoc,
		TypeKind_Object,
		TypeKind_ViewRecord:
		ops = set.From(OperationKind_Insert, OperationKind_Update, OperationKind_Select)
	case TypeKind_Command, TypeKind_Query:
		ops = set.From(OperationKind_Execute)
	case TypeKind_Role:
		ops = set.From(OperationKind_Inherits)
	}
	return ops
}

// isCompatibleOperations returns true if specified operations set contains compatible operations.
func isCompatibleOperations(ops set.Set[OperationKind]) (bool, error) {
	op, ok := ops.First()
	if !ok {
		return false, ErrMissed("operations")
	}

	for o := range ops.Values() {
		if !op.IsCompatible(o) {
			return false, ErrIncompatible("operations %v and %v", op, o)
		}
	}

	return true, nil
}

// Returns true if specified operation is compatible with this operation.
func (k OperationKind) IsCompatible(o OperationKind) bool {
	switch k {
	case OperationKind_Insert, OperationKind_Update, OperationKind_Select:
		return (o == OperationKind_Insert) || (o == OperationKind_Update) || (o == OperationKind_Select)
	case OperationKind_Execute:
		return o == OperationKind_Execute
	case OperationKind_Inherits:
		return o == OperationKind_Inherits
	default:
		panic(ErrUnsupported("operation %v", k))
	}
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
func validateFieldNamesByTypes(tt FindType, types []QName, fields []FieldName) (err error) {
	names := QNamesFrom(types...)

	allFields := map[FieldName]struct{}{}

	for _, n := range names {
		t := tt(n)
		switch t.Kind() {
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
