/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strings"

	"github.com/voedger/voedger/pkg/goutils/set"
)

// Returns all available ACL operations for specified type.
//
// If type can not to be used then returns empty slice.
func ACLOperationsForType(t TypeKind) (ops set.Set[OperationKind]) {
	switch t {
	case TypeKind_GRecord, TypeKind_GDoc,
		TypeKind_CRecord, TypeKind_CDoc,
		TypeKind_WRecord, TypeKind_WDoc,
		TypeKind_ORecord, TypeKind_ODoc,
		TypeKind_Object:
		ops = RecordsOperations
	case TypeKind_ViewRecord:
		ops = ViewRecordsOperations
	case TypeKind_Command, TypeKind_Query:
		ops = set.From(OperationKind_Execute)
	case TypeKind_Role:
		ops = set.From(OperationKind_Inherits)
	}
	return ops
}

// IsCompatibleOperations returns true if specified operations set contains compatible operations.
func IsCompatibleOperations(ops set.Set[OperationKind]) (bool, error) {
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
	if RecordsOperations.Contains(k) {
		return RecordsOperations.Contains(o)
	}
	return k == o
}

// Renders an OperationKind in human-readable form, without "OperationKind_" prefix,
// suitable for debugging or error messages
func (k OperationKind) TrimString() string {
	const pref = "OperationKind_"
	return strings.TrimPrefix(k.String(), pref)
}
