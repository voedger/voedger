/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

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
	// Used to describe projectors.
	OperationKind_Activate

	// Deactivate records. Operation applicable on records.
	// Used to describe projectors.
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
