/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "github.com/voedger/voedger/pkg/goutils/set"

// Enumeration of operation kinds.
//
// Used for ACL and for Limits.
type OperationKind uint8

//go:generate stringer -type=OperationKind -output=stringer_operationkind.go

const (
	OperationKind_null OperationKind = iota

	// Insert records or view records. Operation applicable on records, view records.
	// Used to describe ACL rules, limits and projectors.
	OperationKind_Insert

	// Update records or view records. Operation applicable on records, view records.
	// Used to describe ACL rules, limits and projectors.
	OperationKind_Update

	// Activate records. Operation applicable on records.
	// Used to describe ACL rules, limits and projectors.
	OperationKind_Activate

	// Deactivate records. Operation applicable on records.
	// Used to describe ACL rules, limits and projectors.
	OperationKind_Deactivate

	// Select records or view records. Operation applicable on records, view records.
	// Used to describe ACL rules, limits and projectors.
	OperationKind_Select

	// Execute functions. Operation applicable on commands, queries.
	// Used to describe ACL rules, limits and projectors.
	OperationKind_Execute

	// Parameter for functions. Operation applicable on objects and ODocs.
	// Used to describe projectors.
	OperationKind_ExecuteWithParam

	// Inherit ACL rules from other roles. Operation applicable on roles only.
	// Used to describe ACL rules only.
	OperationKind_Inherits

	OperationKind_count
)

type OperationsSet = set.Set[OperationKind]

// RecordsOperations is a set of operations that applicable on records.
//
// # Contains:
//	 - Insert
//	 - Update
//	 - Activate
//	 - Deactivate
//	 - Select
var RecordsOperations = func() OperationsSet {
	s := set.From(OperationKind_Insert, OperationKind_Update, OperationKind_Activate, OperationKind_Deactivate, OperationKind_Select)
	s.SetReadOnly()
	return s
}()

// ViewRecordsOperations is a set of operations that applicable on view.
//
// # Contains:
//	 - Insert
//	 - Update
//	 - Select
var ViewRecordsOperations = func() OperationsSet {
	s := set.From(OperationKind_Insert, OperationKind_Update, OperationKind_Select)
	s.SetReadOnly()
	return s
}()

// ACL operations is a set of operations that applicable with ACL rules.
//
// # Contains:
//	- RecordsOperations (Insert, Update, Activate, Deactivate, Select)
//	- Execute
//	- Inherits
var ACLOperations = func() OperationsSet {
	s := set.Collect(RecordsOperations.Values())
	s.Set(OperationKind_Execute, OperationKind_Inherits)
	s.SetReadOnly()
	return s
}()

// Limitable operations is a set of operations that can be limited.
//
// # Contains:
//	- RecordsOperations (Insert, Update, Activate, Deactivate, Select)
//	- Execute
var LimitableOperations = func() OperationsSet {
	s := set.Collect(RecordsOperations.Values())
	s.Set(OperationKind_Execute)
	s.SetReadOnly()
	return s
}()

// Projector operations is a set of operations that can trigger a projector.
//
// # Contains:
//	- RecordsOperations (Insert, Update, Activate, Deactivate, Select)
//	- Execute
//	- ExecuteWithParam
var ProjectorOperations = func() OperationsSet {
	s := set.Collect(RecordsOperations.Values())
	s.Set(OperationKind_Execute, OperationKind_ExecuteWithParam)
	s.SetReadOnly()
	return s
}()
