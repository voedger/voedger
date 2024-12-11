/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strings"

	"github.com/voedger/voedger/pkg/goutils/set"
)

// Returns "grant" if policy is allow, "revoke" if deny
func (p PolicyKind) ActionString() string {
	switch p {
	case PolicyKind_Allow:
		return "GRANT"
	case PolicyKind_Deny:
		return "REVOKE"
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
